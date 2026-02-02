package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
)

// CreateReminderTool creates a new reminder
var CreateReminderTool = agent.Tool{
	Name: "create_reminder",
	Description: `Creates a new reminder when a message indicates something the user needs to remember or do.
Use this tool when you detect an actionable task with a due date/time. Examples include:
"Remind me to call mom tomorrow", "Don't forget to submit the report by Friday",
"I need to pick up groceries after work". The reminder should have a clear task
and a determinable due date (explicit or relative).`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"title": agent.PropertyString("Brief actionable title (e.g., 'Call mom', 'Submit report', 'Pick up groceries')"),
		"description": map[string]any{
			"type":        "string",
			"description": "Additional context about the reminder. Optional.",
		},
		"due_date": agent.PropertyString("When the task should be completed, in ISO 8601 format: YYYY-MM-DDTHH:MM:SS"),
		"reminder_time": map[string]any{
			"type":        "string",
			"description": "When to notify the user, in ISO 8601 format. Optional - defaults to due_date.",
		},
		"priority": map[string]any{
			"type":        "string",
			"enum":        []string{"low", "normal", "high"},
			"description": "Priority level of the reminder. Optional - defaults to 'normal'.",
		},
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0 that this is a real reminder"),
		"reasoning":  agent.PropertyString("Brief explanation of why this reminder should be created"),
	}, []string{"title", "due_date", "confidence", "reasoning"}),
}

// UpdateReminderTool updates an existing reminder
var UpdateReminderTool = agent.Tool{
	Name: "update_reminder",
	Description: `Updates an existing reminder when messages indicate changes to a previously created reminder.
Use this tool when someone modifies the due date, title, or details of an existing reminder.
You MUST reference an existing reminder using alfred_reminder_id from the provided context.
Only include fields that are being changed.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"alfred_reminder_id": agent.PropertyInt("Internal Alfred reminder ID for pending reminders (from context)"),
		"title": map[string]any{
			"type":        "string",
			"description": "Updated reminder title. Optional - only if changed.",
		},
		"description": map[string]any{
			"type":        "string",
			"description": "Updated description. Optional - only if changed.",
		},
		"due_date": map[string]any{
			"type":        "string",
			"description": "Updated due date in ISO 8601 format. Optional - only if changed.",
		},
		"reminder_time": map[string]any{
			"type":        "string",
			"description": "Updated reminder notification time in ISO 8601 format. Optional - only if changed.",
		},
		"priority": map[string]any{
			"type":        "string",
			"enum":        []string{"low", "normal", "high"},
			"description": "Updated priority level. Optional - only if changed.",
		},
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0"),
		"reasoning":  agent.PropertyString("Brief explanation of what is being updated and why"),
	}, []string{"alfred_reminder_id", "confidence", "reasoning"}),
}

// DeleteReminderTool cancels/deletes an existing reminder
var DeleteReminderTool = agent.Tool{
	Name: "delete_reminder",
	Description: `Cancels or deletes an existing reminder when messages explicitly indicate cancellation.
Use this tool when someone says a reminder should be cancelled, removed, or is no longer needed.
You MUST reference an existing reminder using alfred_reminder_id from the provided context.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"alfred_reminder_id": agent.PropertyInt("Internal Alfred reminder ID for pending reminders (from context)"),
		"reason":             agent.PropertyString("Brief explanation of why the reminder is being deleted"),
		"confidence":         agent.PropertyNumber("Confidence score from 0.0 to 1.0"),
	}, []string{"alfred_reminder_id", "reason", "confidence"}),
}

// NoReminderActionTool indicates no reminder action is needed
var NoReminderActionTool = agent.Tool{
	Name: "no_reminder_action",
	Description: `Indicates that no reminder action is needed for the analyzed messages.
Use this tool when messages:
- Don't contain actionable tasks or things to remember
- Describe scheduled events/meetings (those are handled by the event analyzer)
- Are general chat without reminder implications
- Have no determinable due date
Always provide reasoning to explain why no action was taken.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"reasoning":  agent.PropertyString("Detailed explanation of why no reminder action is needed"),
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0 that no action is correct"),
	}, []string{"reasoning", "confidence"}),
}

// CreateReminderInput represents parsed input for create_reminder
type CreateReminderInput struct {
	Title        string  `json:"title"`
	Description  string  `json:"description,omitempty"`
	DueDate      string  `json:"due_date"`
	ReminderTime string  `json:"reminder_time,omitempty"`
	Priority     string  `json:"priority,omitempty"`
	Confidence   float64 `json:"confidence"`
	Reasoning    string  `json:"reasoning"`
}

// UpdateReminderInput represents parsed input for update_reminder
type UpdateReminderInput struct {
	AlfredReminderID int64   `json:"alfred_reminder_id"`
	Title            string  `json:"title,omitempty"`
	Description      string  `json:"description,omitempty"`
	DueDate          string  `json:"due_date,omitempty"`
	ReminderTime     string  `json:"reminder_time,omitempty"`
	Priority         string  `json:"priority,omitempty"`
	Confidence       float64 `json:"confidence"`
	Reasoning        string  `json:"reasoning"`
}

// DeleteReminderInput represents parsed input for delete_reminder
type DeleteReminderInput struct {
	AlfredReminderID int64   `json:"alfred_reminder_id"`
	Reason           string  `json:"reason"`
	Confidence       float64 `json:"confidence"`
}

// NoReminderActionInput represents parsed input for no_reminder_action
type NoReminderActionInput struct {
	Reasoning  string  `json:"reasoning"`
	Confidence float64 `json:"confidence"`
}

// HandleCreateReminder processes the create_reminder tool call
func HandleCreateReminder(_ context.Context, input map[string]any) (string, error) {
	parsed := CreateReminderInput{}

	if v, ok := input["title"].(string); ok {
		parsed.Title = v
	}
	if v, ok := input["description"].(string); ok {
		parsed.Description = v
	}
	if v, ok := input["due_date"].(string); ok {
		parsed.DueDate = v
	}
	if v, ok := input["reminder_time"].(string); ok {
		parsed.ReminderTime = v
	}
	if v, ok := input["priority"].(string); ok {
		parsed.Priority = v
	} else {
		parsed.Priority = "normal" // default
	}
	if v, ok := input["confidence"].(float64); ok {
		parsed.Confidence = v
	}
	if v, ok := input["reasoning"].(string); ok {
		parsed.Reasoning = v
	}

	// Validate required fields
	if parsed.Title == "" {
		return "", fmt.Errorf("title is required")
	}
	if parsed.DueDate == "" {
		return "", fmt.Errorf("due_date is required")
	}

	result, err := json.Marshal(map[string]any{
		"status":   "success",
		"action":   "create",
		"reminder": parsed,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// HandleUpdateReminder processes the update_reminder tool call
func HandleUpdateReminder(_ context.Context, input map[string]any) (string, error) {
	parsed := UpdateReminderInput{}

	if v, ok := input["alfred_reminder_id"].(float64); ok {
		parsed.AlfredReminderID = int64(v)
	}
	if v, ok := input["title"].(string); ok {
		parsed.Title = v
	}
	if v, ok := input["description"].(string); ok {
		parsed.Description = v
	}
	if v, ok := input["due_date"].(string); ok {
		parsed.DueDate = v
	}
	if v, ok := input["reminder_time"].(string); ok {
		parsed.ReminderTime = v
	}
	if v, ok := input["priority"].(string); ok {
		parsed.Priority = v
	}
	if v, ok := input["confidence"].(float64); ok {
		parsed.Confidence = v
	}
	if v, ok := input["reasoning"].(string); ok {
		parsed.Reasoning = v
	}

	// Validate - must reference an existing reminder
	if parsed.AlfredReminderID == 0 {
		return "", fmt.Errorf("alfred_reminder_id is required")
	}

	result, err := json.Marshal(map[string]any{
		"status":   "success",
		"action":   "update",
		"reminder": parsed,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// HandleDeleteReminder processes the delete_reminder tool call
func HandleDeleteReminder(_ context.Context, input map[string]any) (string, error) {
	parsed := DeleteReminderInput{}

	if v, ok := input["alfred_reminder_id"].(float64); ok {
		parsed.AlfredReminderID = int64(v)
	}
	if v, ok := input["reason"].(string); ok {
		parsed.Reason = v
	}
	if v, ok := input["confidence"].(float64); ok {
		parsed.Confidence = v
	}

	// Validate - must reference an existing reminder
	if parsed.AlfredReminderID == 0 {
		return "", fmt.Errorf("alfred_reminder_id is required")
	}
	if parsed.Reason == "" {
		return "", fmt.Errorf("reason is required")
	}

	result, err := json.Marshal(map[string]any{
		"status":   "success",
		"action":   "delete",
		"reminder": parsed,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// HandleNoReminderAction processes the no_reminder_action tool call
func HandleNoReminderAction(_ context.Context, input map[string]any) (string, error) {
	parsed := NoReminderActionInput{}

	if v, ok := input["reasoning"].(string); ok {
		parsed.Reasoning = v
	}
	if v, ok := input["confidence"].(float64); ok {
		parsed.Confidence = v
	}

	if parsed.Reasoning == "" {
		return "", fmt.Errorf("reasoning is required")
	}

	result, err := json.Marshal(map[string]any{
		"status":     "success",
		"action":     "none",
		"reasoning":  parsed.Reasoning,
		"confidence": parsed.Confidence,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// AllReminderTools returns all reminder-related tools
func AllReminderTools() []agent.Tool {
	return []agent.Tool{
		CreateReminderTool,
		UpdateReminderTool,
		DeleteReminderTool,
		NoReminderActionTool,
	}
}
