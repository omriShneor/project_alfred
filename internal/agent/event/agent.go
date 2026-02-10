package event

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/agent/langpolicy"
	"github.com/omriShneor/project_alfred/internal/agent/tools"
	"github.com/omriShneor/project_alfred/internal/database"
)

// Agent handles event detection from messages using tool calling
type Agent struct {
	*agent.Agent
}

// Config configures the event agent
type Config struct {
	APIKey      string
	Model       string
	Temperature float64
}

// NewAgent creates a new event scheduling agent
func NewAgent(cfg Config) *Agent {
	baseAgent := agent.NewAgent(agent.AgentConfig{
		Name:         "event-scheduler",
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Temperature:  cfg.Temperature,
		SystemPrompt: EventAnalyzerSystemPrompt,
	})

	// Register extraction tools
	baseAgent.MustRegisterTool(tools.ExtractDateTimeTool, tools.HandleExtractDateTime)
	baseAgent.MustRegisterTool(tools.ExtractLocationTool, tools.HandleExtractLocation)
	baseAgent.MustRegisterTool(tools.ExtractAttendeesTool, tools.HandleExtractAttendees)

	// Register calendar action tools
	baseAgent.MustRegisterTool(tools.CreateCalendarEventTool, tools.HandleCreateCalendarEvent)
	baseAgent.MustRegisterTool(tools.UpdateCalendarEventTool, tools.HandleUpdateCalendarEvent)
	baseAgent.MustRegisterTool(tools.DeleteCalendarEventTool, tools.HandleDeleteCalendarEvent)
	baseAgent.MustRegisterTool(tools.NoActionTool, tools.HandleNoAction)

	return &Agent{Agent: baseAgent}
}

// AnalyzeMessages analyzes chat messages for calendar events
// This method provides backward compatibility with the existing claude.Client interface
func (a *Agent) AnalyzeMessages(
	ctx context.Context,
	history []database.MessageRecord,
	newMessage database.MessageRecord,
	existingEvents []database.CalendarEvent,
) (*agent.EventAnalysis, error) {
	targetLanguage := langpolicy.DetectTargetLanguage(newMessage.MessageText)
	languageInstruction := langpolicy.BuildLanguageInstruction(targetLanguage)
	if targetLanguage.Reliable {
		fmt.Printf(
			"LanguagePolicy[event]: target=%s script=%s confidence=%.2f source=message\n",
			targetLanguage.Code,
			targetLanguage.Script,
			targetLanguage.Confidence,
		)
	}

	analysis, err := a.executePromptAndParse(ctx, buildUserPrompt(
		history,
		newMessage,
		existingEvents,
		languageInstruction,
		"",
	))
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	return a.enforceLanguagePolicy(ctx, targetLanguage, analysis, func(correction string) string {
		return buildUserPrompt(
			history,
			newMessage,
			existingEvents,
			languageInstruction,
			correction,
		)
	})
}

// AnalyzeEmail analyzes an email for calendar events
// Implements agent.EventAnalyzer interface
func (a *Agent) AnalyzeEmail(ctx context.Context, email agent.EmailContent) (*agent.EventAnalysis, error) {
	targetLanguage := langpolicy.DetectTargetLanguage(email.Body)
	languageInstruction := langpolicy.BuildLanguageInstruction(targetLanguage)
	if targetLanguage.Reliable {
		fmt.Printf(
			"LanguagePolicy[event]: target=%s script=%s confidence=%.2f source=email\n",
			targetLanguage.Code,
			targetLanguage.Script,
			targetLanguage.Confidence,
		)
	}

	analysis, err := a.executePromptAndParse(ctx, buildEmailPrompt(email, languageInstruction, ""))
	if err != nil {
		return nil, fmt.Errorf("email analysis failed: %w", err)
	}

	return a.enforceLanguagePolicy(ctx, targetLanguage, analysis, func(correction string) string {
		return buildEmailPrompt(email, languageInstruction, correction)
	})
}

// IsConfigured returns true if the agent is properly configured
func (a *Agent) IsConfigured() bool {
	return a.Agent.IsConfigured()
}

// buildUserPrompt constructs the prompt with message history and context
func buildUserPrompt(
	history []database.MessageRecord,
	newMessage database.MessageRecord,
	existingEvents []database.CalendarEvent,
	languageInstruction string,
	retryInstruction string,
) string {
	var prompt bytes.Buffer

	prompt.WriteString("## Message History (last messages from this channel)\n\n")

	for _, msg := range history {
		if msg.ID == newMessage.ID {
			// Avoid duplicating the trigger message if history already includes it.
			continue
		}
		prompt.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			msg.Timestamp.Format("2006-01-02 15:04"),
			msg.SenderName,
			msg.MessageText,
		))
	}

	prompt.WriteString("\n## New Message (just received)\n\n")
	prompt.WriteString(fmt.Sprintf("[%s] %s: %s\n",
		newMessage.Timestamp.Format("2006-01-02 15:04"),
		newMessage.SenderName,
		newMessage.MessageText,
	))

	if len(existingEvents) > 0 {
		prompt.WriteString("\n## Existing Calendar Events for this channel\n\n")
		for _, event := range existingEvents {
			endStr := ""
			if event.EndTime != nil {
				endStr = fmt.Sprintf(" - %s", event.EndTime.Format("2006-01-02 15:04"))
			}
			googleID := "none"
			if event.GoogleEventID != nil && *event.GoogleEventID != "" {
				googleID = *event.GoogleEventID
			}
			prompt.WriteString(fmt.Sprintf("- [AlfredID: %d, GoogleID: %s, Status: %s] %s @ %s%s",
				event.ID,
				googleID,
				event.Status,
				event.Title,
				event.StartTime.Format("2006-01-02 15:04"),
				endStr,
			))
			if event.Location != "" {
				prompt.WriteString(fmt.Sprintf(" (Location: %s)", event.Location))
			}
			prompt.WriteString("\n")
		}
	} else {
		prompt.WriteString("\n## Existing Calendar Events for this channel\n\nNo existing events.\n")
	}

	prompt.WriteString("\n## Current Date/Time Reference\n\n")
	now := time.Now()
	prompt.WriteString(fmt.Sprintf("Current time: %s (%s)\n", now.Format("2006-01-02 15:04:05 Monday -07:00"), now.Location().String()))

	if languageInstruction != "" {
		prompt.WriteString("\n## Output Language Requirement\n\n")
		prompt.WriteString(languageInstruction + "\n")
	}
	if retryInstruction != "" {
		prompt.WriteString("\n## Correction Required\n\n")
		prompt.WriteString(retryInstruction + "\n")
	}

	prompt.WriteString("\nAnalyze these messages using the available tools. First extract relevant information, then take the appropriate calendar action.")

	return prompt.String()
}

// buildEmailPrompt constructs the prompt for email analysis
func buildEmailPrompt(email agent.EmailContent, languageInstruction, retryInstruction string) string {
	var prompt bytes.Buffer

	// Add thread history if present
	if len(email.ThreadHistory) > 0 {
		prompt.WriteString("## Email Thread History (chronological order)\n\n")
		for _, msg := range email.ThreadHistory {
			prompt.WriteString(fmt.Sprintf("[%s] From: %s\n", msg.Date, msg.From))
			prompt.WriteString(fmt.Sprintf("Subject: %s\n", msg.Subject))
			prompt.WriteString(fmt.Sprintf("Body:\n%s\n\n---\n\n", truncateBody(msg.Body, 2000)))
		}
	}

	prompt.WriteString("## Email to Analyze (latest in thread)\n\n")
	prompt.WriteString(fmt.Sprintf("**From:** %s\n", email.From))
	prompt.WriteString(fmt.Sprintf("**To:** %s\n", email.To))
	prompt.WriteString(fmt.Sprintf("**Date:** %s\n", email.Date))
	prompt.WriteString(fmt.Sprintf("**Subject:** %s\n\n", email.Subject))
	prompt.WriteString("**Body:**\n")
	prompt.WriteString(truncateBody(email.Body, 8000))

	prompt.WriteString("\n\n## Current Date/Time Reference\n\n")
	now := time.Now()
	prompt.WriteString(fmt.Sprintf("Current time: %s (%s)\n", now.Format("2006-01-02 15:04:05 Monday -07:00"), now.Location().String()))

	if languageInstruction != "" {
		prompt.WriteString("\n## Output Language Requirement\n\n")
		prompt.WriteString(languageInstruction + "\n")
	}
	if retryInstruction != "" {
		prompt.WriteString("\n## Correction Required\n\n")
		prompt.WriteString(retryInstruction + "\n")
	}

	prompt.WriteString("\nAnalyze this email using the available tools. First extract relevant information, then take the appropriate calendar action.")

	return prompt.String()
}

func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "\n\n[... content truncated ...]"
}

func (a *Agent) executePromptAndParse(ctx context.Context, userPrompt string) (*agent.EventAnalysis, error) {
	input := agent.AgentInput{
		Messages: []agent.Message{
			{
				Role: "user",
				Content: []agent.ContentBlock{
					agent.TextBlock{Type: "text", Text: userPrompt},
				},
			},
		},
		MaxTurns: 6, // Allow extraction + action + final response
	}

	output, err := a.Execute(ctx, input)
	if err != nil {
		return nil, err
	}

	return parseAgentOutput(output)
}

func (a *Agent) enforceLanguagePolicy(
	ctx context.Context,
	target langpolicy.TargetLanguage,
	initial *agent.EventAnalysis,
	retryPromptBuilder func(correction string) string,
) (*agent.EventAnalysis, error) {
	shouldRetry, validation := shouldRetryEventForLanguage(target, initial)
	if !shouldRetry {
		if target.Reliable {
			fmt.Printf(
				"LanguagePolicy[event]: validation=pass action=%s checked=%d skipped=%d\n",
				initial.Action,
				validation.CheckedFields,
				validation.SkippedFields,
			)
		}
		return initial, nil
	}

	fmt.Printf(
		"LanguagePolicy[event]: validation=fail action=%s mismatches=%s retry=true\n",
		initial.Action,
		formatMismatches(validation),
	)

	retryPrompt := retryPromptBuilder(langpolicy.BuildCorrectiveRetryInstruction(target, validation))
	retryAnalysis, err := a.executePromptAndParse(ctx, retryPrompt)
	if err != nil {
		fmt.Printf("LanguagePolicy[event]: retry_error=%v fallback=initial\n", err)
		return initial, nil
	}

	retryNeeded, retryValidation := shouldRetryEventForLanguage(target, retryAnalysis)
	if !retryNeeded {
		fmt.Printf(
			"LanguagePolicy[event]: retry_result=pass action=%s checked=%d skipped=%d\n",
			retryAnalysis.Action,
			retryValidation.CheckedFields,
			retryValidation.SkippedFields,
		)
		return retryAnalysis, nil
	}

	fmt.Printf(
		"LanguagePolicy[event]: retry_result=fail action=%s mismatches=%s fallback=retry\n",
		retryAnalysis.Action,
		formatMismatches(retryValidation),
	)
	return retryAnalysis, nil
}

func shouldRetryEventForLanguage(
	target langpolicy.TargetLanguage,
	analysis *agent.EventAnalysis,
) (bool, langpolicy.ValidationResult) {
	empty := langpolicy.ValidationResult{}

	if analysis == nil || analysis.Event == nil {
		return false, empty
	}
	if !target.Reliable || target.Code == "" {
		return false, empty
	}
	if analysis.Action != "create" && analysis.Action != "update" {
		return false, empty
	}

	validation := langpolicy.ValidateFieldsLanguage(target, map[string]string{
		"title":       analysis.Event.Title,
		"description": analysis.Event.Description,
		"location":    analysis.Event.Location,
	})
	return !validation.IsMatch(), validation
}

func formatMismatches(validation langpolicy.ValidationResult) string {
	if len(validation.Mismatches) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(validation.Mismatches))
	for _, mismatch := range validation.Mismatches {
		parts = append(parts, fmt.Sprintf("%s(%s)", mismatch.Field, mismatch.DetectedCode))
	}
	return strings.Join(parts, ", ")
}

// parseAgentOutput converts agent output to EventAnalysis
func parseAgentOutput(output *agent.AgentOutput) (*agent.EventAnalysis, error) {
	if len(output.ToolCalls) == 0 {
		return &agent.EventAnalysis{
			HasEvent:   false,
			Action:     "none",
			Reasoning:  "No tools were called",
			Confidence: 0,
		}, nil
	}

	// Validate terminal action semantics: exactly one action tool call.
	actionCalls := make([]*agent.ToolCall, 0, 1)
	for i := range output.ToolCalls {
		call := &output.ToolCalls[i]
		switch call.Name {
		case "create_calendar_event", "update_calendar_event", "delete_calendar_event", "no_calendar_action":
			actionCalls = append(actionCalls, call)
		}
	}

	if len(actionCalls) == 0 {
		return &agent.EventAnalysis{
			HasEvent:   false,
			Action:     "none",
			Reasoning:  "No action tool was called",
			Confidence: 0,
		}, nil
	}
	if len(actionCalls) > 1 {
		return &agent.EventAnalysis{
			HasEvent:   false,
			Action:     "none",
			Reasoning:  "Ambiguous tool output: multiple action tools called",
			Confidence: 0,
		}, nil
	}
	actionCall := actionCalls[0]
	if actionCall.Error != nil {
		return &agent.EventAnalysis{
			HasEvent:   false,
			Action:     "none",
			Reasoning:  fmt.Sprintf("Action tool failed: %v", actionCall.Error),
			Confidence: 0,
		}, nil
	}

	// Parse the action result
	var result map[string]any
	if err := json.Unmarshal([]byte(actionCall.Output), &result); err != nil {
		return nil, fmt.Errorf("failed to parse action result: %w", err)
	}

	action, _ := result["action"].(string)

	analysis := &agent.EventAnalysis{
		HasEvent: action != "none",
		Action:   action,
	}

	// Extract event data based on action type
	switch action {
	case "create":
		if eventData, ok := result["event"].(map[string]any); ok {
			analysis.Event = parseEventData(eventData)
			if conf, ok := eventData["confidence"].(float64); ok {
				analysis.Confidence = conf
			}
			if reason, ok := eventData["reasoning"].(string); ok {
				analysis.Reasoning = reason
			}
		}
	case "update":
		if eventData, ok := result["event"].(map[string]any); ok {
			analysis.Event = parseEventData(eventData)
			if conf, ok := eventData["confidence"].(float64); ok {
				analysis.Confidence = conf
			}
			if reason, ok := eventData["reasoning"].(string); ok {
				analysis.Reasoning = reason
			}
		}
	case "delete":
		if eventData, ok := result["event"].(map[string]any); ok {
			analysis.Event = &agent.EventData{}
			if alfredID, ok := eventData["alfred_event_id"].(float64); ok {
				analysis.Event.AlfredEventRef = int64(alfredID)
			}
			if googleID, ok := eventData["google_event_id"].(string); ok {
				analysis.Event.UpdateRef = googleID
			}
			if conf, ok := eventData["confidence"].(float64); ok {
				analysis.Confidence = conf
			}
			if reason, ok := eventData["reason"].(string); ok {
				analysis.Reasoning = reason
			}
		}
	case "none":
		if reason, ok := result["reasoning"].(string); ok {
			analysis.Reasoning = reason
		}
		if conf, ok := result["confidence"].(float64); ok {
			analysis.Confidence = conf
		}
	}

	return analysis, nil
}

// parseEventData extracts EventData from a map
func parseEventData(data map[string]any) *agent.EventData {
	event := &agent.EventData{}

	if v, ok := data["title"].(string); ok {
		event.Title = v
	}
	if v, ok := data["description"].(string); ok {
		event.Description = v
	}
	if v, ok := data["start_time"].(string); ok {
		event.StartTime = v
	}
	if v, ok := data["end_time"].(string); ok {
		event.EndTime = v
	}
	if v, ok := data["location"].(string); ok {
		event.Location = v
	}
	if v, ok := data["alfred_event_id"].(float64); ok {
		event.AlfredEventRef = int64(v)
	}
	if v, ok := data["google_event_id"].(string); ok {
		event.UpdateRef = v
	}
	if attendeesRaw, ok := data["attendees"].([]any); ok {
		for _, a := range attendeesRaw {
			aMap, ok := a.(map[string]any)
			if !ok {
				continue
			}

			attendee := agent.EventAttendeeData{}
			if v, ok := aMap["name"].(string); ok {
				attendee.Name = v
			}
			if v, ok := aMap["email"].(string); ok {
				attendee.Email = v
			}
			if v, ok := aMap["phone"].(string); ok {
				attendee.Phone = v
			}
			if v, ok := aMap["role"].(string); ok {
				attendee.Role = v
			}
			event.Attendees = append(event.Attendees, attendee)
		}
	}

	return event
}
