package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
)

// CreateCalendarEventTool creates a new calendar event
var CreateCalendarEventTool = agent.Tool{
	Name: "create_calendar_event",
	Description: `Creates a new calendar event when a message clearly indicates a scheduled activity.
Use this tool when you detect a new event that should be added to the user's calendar.
The event should have a specific date and time, either explicit ("January 15th at 3pm")
or relative to current time ("tomorrow at noon", "next Tuesday"). Do NOT create events
for vague mentions without actionable scheduling details. Include all relevant details
extracted from the message context.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"title": agent.PropertyString("Concise, descriptive event title (e.g., 'Team Meeting', 'Lunch with Sarah')"),
		"description": map[string]any{
			"type":        "string",
			"description": "Additional context from the messages. Optional.",
		},
		"start_time": agent.PropertyString("Event start time in ISO 8601 format: YYYY-MM-DDTHH:MM:SS"),
		"end_time": map[string]any{
			"type":        "string",
			"description": "Event end time in ISO 8601 format. Optional - defaults to 1 hour after start.",
		},
		"location": map[string]any{
			"type":        "string",
			"description": "Event location if mentioned. Optional.",
		},
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0 that this is a real event"),
		"reasoning": agent.PropertyString("Brief explanation of why this event should be created"),
	}, []string{"title", "start_time", "confidence", "reasoning"}),
}

// UpdateCalendarEventTool updates an existing calendar event
var UpdateCalendarEventTool = agent.Tool{
	Name: "update_calendar_event",
	Description: `Updates an existing calendar event when messages indicate changes to a previously scheduled activity.
Use this tool when someone modifies the time, date, location, or details of an existing event.
You MUST reference an existing event from the provided context. For events already synced
to Google Calendar, use google_event_id. For events still pending review in Alfred, use
alfred_event_id. Only include fields that are being changed.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"alfred_event_id": map[string]any{
			"type":        "integer",
			"description": "Internal Alfred event ID for pending events (from context)",
		},
		"google_event_id": map[string]any{
			"type":        "string",
			"description": "Google Calendar event ID for synced events (from context)",
		},
		"title": map[string]any{
			"type":        "string",
			"description": "Updated event title. Optional - only if changed.",
		},
		"description": map[string]any{
			"type":        "string",
			"description": "Updated description. Optional - only if changed.",
		},
		"start_time": map[string]any{
			"type":        "string",
			"description": "Updated start time in ISO 8601 format. Optional - only if changed.",
		},
		"end_time": map[string]any{
			"type":        "string",
			"description": "Updated end time in ISO 8601 format. Optional - only if changed.",
		},
		"location": map[string]any{
			"type":        "string",
			"description": "Updated location. Optional - only if changed.",
		},
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0"),
		"reasoning": agent.PropertyString("Brief explanation of what is being updated and why"),
	}, []string{"confidence", "reasoning"}),
}

// DeleteCalendarEventTool cancels/deletes an existing calendar event
var DeleteCalendarEventTool = agent.Tool{
	Name: "delete_calendar_event",
	Description: `Cancels or deletes an existing calendar event when messages explicitly indicate cancellation.
Use this tool when someone says an event is cancelled, no longer happening, or should be removed.
You MUST reference an existing event from the provided context. For synced events, use
google_event_id. For pending events, use alfred_event_id.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"alfred_event_id": map[string]any{
			"type":        "integer",
			"description": "Internal Alfred event ID for pending events (from context)",
		},
		"google_event_id": map[string]any{
			"type":        "string",
			"description": "Google Calendar event ID for synced events (from context)",
		},
		"reason": agent.PropertyString("Brief explanation of why the event is being deleted"),
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0"),
	}, []string{"reason", "confidence"}),
}

// NoActionTool indicates no calendar action is needed
var NoActionTool = agent.Tool{
	Name: "no_calendar_action",
	Description: `Indicates that no calendar action is needed for the analyzed messages.
Use this tool when messages don't contain scheduling information, are general chat,
discuss past events, mention events without clear scheduling intent, or are too vague
to create a calendar entry. Always provide reasoning to explain why no action was taken.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"reasoning": agent.PropertyString("Detailed explanation of why no calendar action is needed"),
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0 that no action is correct"),
	}, []string{"reasoning", "confidence"}),
}

// CreateEventInput represents parsed input for create_calendar_event
type CreateEventInput struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time,omitempty"`
	Location    string  `json:"location,omitempty"`
	Confidence  float64 `json:"confidence"`
	Reasoning   string  `json:"reasoning"`
}

// UpdateEventInput represents parsed input for update_calendar_event
type UpdateEventInput struct {
	AlfredEventID  int64   `json:"alfred_event_id,omitempty"`
	GoogleEventID  string  `json:"google_event_id,omitempty"`
	Title          string  `json:"title,omitempty"`
	Description    string  `json:"description,omitempty"`
	StartTime      string  `json:"start_time,omitempty"`
	EndTime        string  `json:"end_time,omitempty"`
	Location       string  `json:"location,omitempty"`
	Confidence     float64 `json:"confidence"`
	Reasoning      string  `json:"reasoning"`
}

// DeleteEventInput represents parsed input for delete_calendar_event
type DeleteEventInput struct {
	AlfredEventID int64   `json:"alfred_event_id,omitempty"`
	GoogleEventID string  `json:"google_event_id,omitempty"`
	Reason        string  `json:"reason"`
	Confidence    float64 `json:"confidence"`
}

// NoActionInput represents parsed input for no_calendar_action
type NoActionInput struct {
	Reasoning  string  `json:"reasoning"`
	Confidence float64 `json:"confidence"`
}

// HandleCreateCalendarEvent processes the create_calendar_event tool call
func HandleCreateCalendarEvent(_ context.Context, input map[string]any) (string, error) {
	parsed := CreateEventInput{}

	if v, ok := input["title"].(string); ok {
		parsed.Title = v
	}
	if v, ok := input["description"].(string); ok {
		parsed.Description = v
	}
	if v, ok := input["start_time"].(string); ok {
		parsed.StartTime = v
	}
	if v, ok := input["end_time"].(string); ok {
		parsed.EndTime = v
	}
	if v, ok := input["location"].(string); ok {
		parsed.Location = v
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
	if parsed.StartTime == "" {
		return "", fmt.Errorf("start_time is required")
	}

	result, err := json.Marshal(map[string]any{
		"status": "success",
		"action": "create",
		"event":  parsed,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// HandleUpdateCalendarEvent processes the update_calendar_event tool call
func HandleUpdateCalendarEvent(_ context.Context, input map[string]any) (string, error) {
	parsed := UpdateEventInput{}

	if v, ok := input["alfred_event_id"].(float64); ok {
		parsed.AlfredEventID = int64(v)
	}
	if v, ok := input["google_event_id"].(string); ok {
		parsed.GoogleEventID = v
	}
	if v, ok := input["title"].(string); ok {
		parsed.Title = v
	}
	if v, ok := input["description"].(string); ok {
		parsed.Description = v
	}
	if v, ok := input["start_time"].(string); ok {
		parsed.StartTime = v
	}
	if v, ok := input["end_time"].(string); ok {
		parsed.EndTime = v
	}
	if v, ok := input["location"].(string); ok {
		parsed.Location = v
	}
	if v, ok := input["confidence"].(float64); ok {
		parsed.Confidence = v
	}
	if v, ok := input["reasoning"].(string); ok {
		parsed.Reasoning = v
	}

	// Validate - must reference an existing event
	if parsed.AlfredEventID == 0 && parsed.GoogleEventID == "" {
		return "", fmt.Errorf("either alfred_event_id or google_event_id is required")
	}

	result, err := json.Marshal(map[string]any{
		"status": "success",
		"action": "update",
		"event":  parsed,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// HandleDeleteCalendarEvent processes the delete_calendar_event tool call
func HandleDeleteCalendarEvent(_ context.Context, input map[string]any) (string, error) {
	parsed := DeleteEventInput{}

	if v, ok := input["alfred_event_id"].(float64); ok {
		parsed.AlfredEventID = int64(v)
	}
	if v, ok := input["google_event_id"].(string); ok {
		parsed.GoogleEventID = v
	}
	if v, ok := input["reason"].(string); ok {
		parsed.Reason = v
	}
	if v, ok := input["confidence"].(float64); ok {
		parsed.Confidence = v
	}

	// Validate - must reference an existing event
	if parsed.AlfredEventID == 0 && parsed.GoogleEventID == "" {
		return "", fmt.Errorf("either alfred_event_id or google_event_id is required")
	}
	if parsed.Reason == "" {
		return "", fmt.Errorf("reason is required")
	}

	result, err := json.Marshal(map[string]any{
		"status": "success",
		"action": "delete",
		"event":  parsed,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// HandleNoAction processes the no_calendar_action tool call
func HandleNoAction(_ context.Context, input map[string]any) (string, error) {
	parsed := NoActionInput{}

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

// AllCalendarTools returns all calendar-related tools
func AllCalendarTools() []agent.Tool {
	return []agent.Tool{
		CreateCalendarEventTool,
		UpdateCalendarEventTool,
		DeleteCalendarEventTool,
		NoActionTool,
	}
}

// AllExtractionTools returns all extraction tools
func AllExtractionTools() []agent.Tool {
	return []agent.Tool{
		ExtractDateTimeTool,
		ExtractLocationTool,
		ExtractAttendeesTool,
	}
}
