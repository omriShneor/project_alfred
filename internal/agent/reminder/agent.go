package reminder

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

// Agent handles reminder detection from messages using tool calling
type Agent struct {
	*agent.Agent
}

// Config configures the reminder agent
type Config struct {
	APIKey      string
	Model       string
	Temperature float64
}

// NewAgent creates a new reminder scheduling agent
func NewAgent(cfg Config) *Agent {
	baseAgent := agent.NewAgent(agent.AgentConfig{
		Name:         "reminder-scheduler",
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		Temperature:  cfg.Temperature,
		SystemPrompt: ReminderAnalyzerSystemPrompt,
	})

	// REUSE extraction tool from event agent
	baseAgent.MustRegisterTool(tools.ExtractDateTimeTool, tools.HandleExtractDateTime)

	// Register reminder-specific action tools
	baseAgent.MustRegisterTool(tools.CreateReminderTool, tools.HandleCreateReminder)
	baseAgent.MustRegisterTool(tools.UpdateReminderTool, tools.HandleUpdateReminder)
	baseAgent.MustRegisterTool(tools.DeleteReminderTool, tools.HandleDeleteReminder)
	baseAgent.MustRegisterTool(tools.NoReminderActionTool, tools.HandleNoReminderAction)

	return &Agent{Agent: baseAgent}
}

// AnalyzeMessages analyzes chat messages for reminders
// Implements agent.ReminderAnalyzer interface
func (a *Agent) AnalyzeMessages(
	ctx context.Context,
	history []database.MessageRecord,
	newMessage database.MessageRecord,
	existingReminders []database.Reminder,
) (*agent.ReminderAnalysis, error) {
	targetLanguage := langpolicy.DetectTargetLanguage(newMessage.MessageText)
	languageInstruction := langpolicy.BuildLanguageInstruction(targetLanguage)
	if targetLanguage.Reliable {
		fmt.Printf(
			"LanguagePolicy[reminder]: target=%s script=%s confidence=%.2f source=message\n",
			targetLanguage.Code,
			targetLanguage.Script,
			targetLanguage.Confidence,
		)
	}

	analysis, err := a.executePromptAndParse(ctx, buildUserPrompt(
		history,
		newMessage,
		existingReminders,
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
			existingReminders,
			languageInstruction,
			correction,
		)
	})
}

// AnalyzeEmail analyzes an email for reminders
// Implements agent.ReminderAnalyzer interface
func (a *Agent) AnalyzeEmail(ctx context.Context, email agent.EmailContent) (*agent.ReminderAnalysis, error) {
	// Detect language from subject+body so a single localized token in the body
	// doesn't incorrectly flip the entire output language.
	targetLanguage := langpolicy.DetectTargetLanguage(strings.TrimSpace(email.Subject + "\n" + email.Body))
	languageInstruction := langpolicy.BuildLanguageInstruction(targetLanguage)
	if targetLanguage.Reliable {
		fmt.Printf(
			"LanguagePolicy[reminder]: target=%s script=%s confidence=%.2f source=email\n",
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
	existingReminders []database.Reminder,
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

	if len(existingReminders) > 0 {
		prompt.WriteString("\n## Existing Reminders for this channel\n\n")
		for _, reminder := range existingReminders {
			dueLabel := "No due date"
			if reminder.DueDate != nil {
				dueLabel = reminder.DueDate.Format("2006-01-02 15:04")
			}
			prompt.WriteString(fmt.Sprintf("- [AlfredID: %d, Status: %s, Priority: %s] %s - Due: %s",
				reminder.ID,
				reminder.Status,
				reminder.Priority,
				reminder.Title,
				dueLabel,
			))
			if reminder.Description != "" {
				prompt.WriteString(fmt.Sprintf(" (%s)", reminder.Description))
			}
			prompt.WriteString("\n")
		}
	} else {
		prompt.WriteString("\n## Existing Reminders for this channel\n\nNo existing reminders.\n")
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

	prompt.WriteString("\nAnalyze these messages using the available tools. First extract relevant date/time information if needed, then take the appropriate reminder action.")

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

	prompt.WriteString("\nAnalyze this email using the available tools. First extract date/time information if needed, then take the appropriate reminder action.")

	return prompt.String()
}

func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "\n\n[... content truncated ...]"
}

func (a *Agent) executePromptAndParse(ctx context.Context, userPrompt string) (*agent.ReminderAnalysis, error) {
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
	initial *agent.ReminderAnalysis,
	retryPromptBuilder func(correction string) string,
) (*agent.ReminderAnalysis, error) {
	shouldRetry, validation := shouldRetryReminderForLanguage(target, initial)
	if !shouldRetry {
		if target.Reliable {
			fmt.Printf(
				"LanguagePolicy[reminder]: validation=pass action=%s checked=%d skipped=%d\n",
				initial.Action,
				validation.CheckedFields,
				validation.SkippedFields,
			)
		}
		return initial, nil
	}

	fmt.Printf(
		"LanguagePolicy[reminder]: validation=fail action=%s mismatches=%s retry=true\n",
		initial.Action,
		formatMismatches(validation),
	)

	retryPrompt := retryPromptBuilder(langpolicy.BuildCorrectiveRetryInstruction(target, validation))
	retryAnalysis, err := a.executePromptAndParse(ctx, retryPrompt)
	if err != nil {
		fmt.Printf("LanguagePolicy[reminder]: retry_error=%v fallback=initial\n", err)
		return initial, nil
	}

	retryNeeded, retryValidation := shouldRetryReminderForLanguage(target, retryAnalysis)
	if !retryNeeded {
		fmt.Printf(
			"LanguagePolicy[reminder]: retry_result=pass action=%s checked=%d skipped=%d\n",
			retryAnalysis.Action,
			retryValidation.CheckedFields,
			retryValidation.SkippedFields,
		)
		return retryAnalysis, nil
	}

	fmt.Printf(
		"LanguagePolicy[reminder]: retry_result=fail action=%s mismatches=%s fallback=retry\n",
		retryAnalysis.Action,
		formatMismatches(retryValidation),
	)
	return retryAnalysis, nil
}

func shouldRetryReminderForLanguage(
	target langpolicy.TargetLanguage,
	analysis *agent.ReminderAnalysis,
) (bool, langpolicy.ValidationResult) {
	empty := langpolicy.ValidationResult{}

	if analysis == nil || analysis.Reminder == nil {
		return false, empty
	}
	if !target.Reliable || target.Code == "" {
		return false, empty
	}
	if analysis.Action != "create" && analysis.Action != "update" {
		return false, empty
	}

	validation := langpolicy.ValidateFieldsLanguage(target, map[string]string{
		"title":       analysis.Reminder.Title,
		"description": analysis.Reminder.Description,
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

// parseAgentOutput converts agent output to ReminderAnalysis
func parseAgentOutput(output *agent.AgentOutput) (*agent.ReminderAnalysis, error) {
	if len(output.ToolCalls) == 0 {
		return &agent.ReminderAnalysis{
			HasReminder: false,
			Action:      "none",
			Reasoning:   "No tools were called",
			Confidence:  0,
		}, nil
	}

	// Validate terminal action semantics: exactly one action tool call.
	actionCalls := make([]*agent.ToolCall, 0, 1)
	for i := range output.ToolCalls {
		call := &output.ToolCalls[i]
		switch call.Name {
		case "create_reminder", "update_reminder", "delete_reminder", "no_reminder_action":
			actionCalls = append(actionCalls, call)
		}
	}

	if len(actionCalls) == 0 {
		return &agent.ReminderAnalysis{
			HasReminder: false,
			Action:      "none",
			Reasoning:   "No action tool was called",
			Confidence:  0,
		}, nil
	}
	if len(actionCalls) > 1 {
		return &agent.ReminderAnalysis{
			HasReminder: false,
			Action:      "none",
			Reasoning:   "Ambiguous tool output: multiple action tools called",
			Confidence:  0,
		}, nil
	}
	actionCall := actionCalls[0]
	if actionCall.Error != nil {
		return &agent.ReminderAnalysis{
			HasReminder: false,
			Action:      "none",
			Reasoning:   fmt.Sprintf("Action tool failed: %v", actionCall.Error),
			Confidence:  0,
		}, nil
	}

	// Parse the action result
	var result map[string]any
	if err := json.Unmarshal([]byte(actionCall.Output), &result); err != nil {
		return nil, fmt.Errorf("failed to parse action result: %w", err)
	}

	action, _ := result["action"].(string)

	analysis := &agent.ReminderAnalysis{
		HasReminder: action != "none",
		Action:      action,
	}

	// Extract reminder data based on action type
	switch action {
	case "create":
		if reminderData, ok := result["reminder"].(map[string]any); ok {
			analysis.Reminder = parseReminderData(reminderData)
			if conf, ok := reminderData["confidence"].(float64); ok {
				analysis.Confidence = conf
			}
			if reason, ok := reminderData["reasoning"].(string); ok {
				analysis.Reasoning = reason
			}
		}
	case "update":
		if reminderData, ok := result["reminder"].(map[string]any); ok {
			analysis.Reminder = parseReminderData(reminderData)
			if conf, ok := reminderData["confidence"].(float64); ok {
				analysis.Confidence = conf
			}
			if reason, ok := reminderData["reasoning"].(string); ok {
				analysis.Reasoning = reason
			}
		}
	case "delete":
		if reminderData, ok := result["reminder"].(map[string]any); ok {
			analysis.Reminder = &agent.ReminderData{}
			if alfredID, ok := reminderData["alfred_reminder_id"].(float64); ok {
				analysis.Reminder.AlfredReminderRef = int64(alfredID)
			}
			if conf, ok := reminderData["confidence"].(float64); ok {
				analysis.Confidence = conf
			}
			if reason, ok := reminderData["reason"].(string); ok {
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

// parseReminderData extracts ReminderData from a map
func parseReminderData(data map[string]any) *agent.ReminderData {
	reminder := &agent.ReminderData{}

	if v, ok := data["title"].(string); ok {
		reminder.Title = v
	}
	if v, ok := data["description"].(string); ok {
		reminder.Description = v
	}
	if v, ok := data["due_date"].(string); ok {
		reminder.DueDate = v
	}
	if v, ok := data["reminder_time"].(string); ok {
		reminder.ReminderTime = v
	}
	if v, ok := data["priority"].(string); ok {
		reminder.Priority = v
	}
	if v, ok := data["alfred_reminder_id"].(float64); ok {
		reminder.AlfredReminderRef = int64(v)
	}

	return reminder
}
