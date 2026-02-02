package event

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
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
		SystemPrompt: SystemPrompt,
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
	userPrompt := buildUserPrompt(history, newMessage, existingEvents)

	input := agent.AgentInput{
		Messages: []agent.Message{
			{
				Role: "user",
				Content: []agent.ContentBlock{
					agent.TextBlock{Type: "text", Text: userPrompt},
				},
			},
		},
		MaxTurns: 2, // Allow extraction + action
	}

	output, err := a.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	return parseAgentOutput(output)
}

// AnalyzeEmail analyzes an email for calendar events
// Implements agent.Analyzer interface
func (a *Agent) AnalyzeEmail(ctx context.Context, email agent.EmailContent) (*agent.EventAnalysis, error) {
	// Create a temporary agent with email-specific prompt
	emailAgent := agent.NewAgent(agent.AgentConfig{
		Name:         "event-scheduler-email",
		APIKey:       "", // Will use same client
		Model:        "",
		Temperature:  0,
		SystemPrompt: EmailSystemPrompt,
	})

	// Copy tools from main agent
	for _, tool := range a.Tools() {
		handler := a.getToolHandler(tool.Name)
		if handler != nil {
			emailAgent.MustRegisterTool(tool, handler)
		}
	}

	userPrompt := buildEmailPrompt(email)

	input := agent.AgentInput{
		Messages: []agent.Message{
			{
				Role: "user",
				Content: []agent.ContentBlock{
					agent.TextBlock{Type: "text", Text: userPrompt},
				},
			},
		},
		MaxTurns: 2,
	}

	// Use the main agent (which has the API client configured)
	output, err := a.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("email analysis failed: %w", err)
	}

	return parseAgentOutput(output)
}

// getToolHandler returns the handler for a tool (used internally)
func (a *Agent) getToolHandler(name string) agent.ToolHandler {
	switch name {
	case "extract_datetime":
		return tools.HandleExtractDateTime
	case "extract_location":
		return tools.HandleExtractLocation
	case "extract_attendees":
		return tools.HandleExtractAttendees
	case "create_calendar_event":
		return tools.HandleCreateCalendarEvent
	case "update_calendar_event":
		return tools.HandleUpdateCalendarEvent
	case "delete_calendar_event":
		return tools.HandleDeleteCalendarEvent
	case "no_calendar_action":
		return tools.HandleNoAction
	default:
		return nil
	}
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
) string {
	var prompt bytes.Buffer

	prompt.WriteString("## Message History (last messages from this channel)\n\n")

	for _, msg := range history {
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
	prompt.WriteString(fmt.Sprintf("Current time: %s\n", time.Now().Format("2006-01-02 15:04 (Monday)")))

	prompt.WriteString("\nAnalyze these messages using the available tools. First extract relevant information, then take the appropriate calendar action.")

	return prompt.String()
}

// buildEmailPrompt constructs the prompt for email analysis
func buildEmailPrompt(email agent.EmailContent) string {
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
	prompt.WriteString(fmt.Sprintf("Current time: %s\n", time.Now().Format("2006-01-02 15:04 (Monday)")))

	prompt.WriteString("\nAnalyze this email using the available tools. First extract relevant information, then take the appropriate calendar action.")

	return prompt.String()
}

func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "\n\n[... content truncated ...]"
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

	// Find the action tool call (the last one that determines the action)
	var actionCall *agent.ToolCall
findAction:
	for i := len(output.ToolCalls) - 1; i >= 0; i-- {
		call := &output.ToolCalls[i]
		switch call.Name {
		case "create_calendar_event", "update_calendar_event", "delete_calendar_event", "no_calendar_action":
			actionCall = call
			break findAction
		}
	}

	if actionCall == nil {
		return &agent.EventAnalysis{
			HasEvent:   false,
			Action:     "none",
			Reasoning:  "No action tool was called",
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

	return event
}
