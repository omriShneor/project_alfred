package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
)

// ExtractDateTimeTool extracts date/time information from text
var ExtractDateTimeTool = agent.Tool{
	Name: "extract_datetime",
	Description: `Extracts date and time information from natural language text for calendar events.
Handles absolute dates ("January 15th at 3pm", "2024-02-14 14:00"), relative dates
("tomorrow", "next Tuesday", "in 2 hours"), and time ranges ("2-4pm", "from 10am to noon").
Returns ISO 8601 formatted datetime strings. Use the current_datetime provided in context
to resolve relative dates. If the text doesn't contain clear scheduling information,
return has_datetime: false with reasoning.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"has_datetime": agent.PropertyBool("Whether the text contains date/time information for scheduling"),
		"start_time": map[string]any{
			"type":        "string",
			"description": "Event start time in ISO 8601 format (YYYY-MM-DDTHH:MM:SS). Required if has_datetime is true.",
		},
		"end_time": map[string]any{
			"type":        "string",
			"description": "Event end time in ISO 8601 format. Optional - omit if not specified in text.",
		},
		"is_all_day": agent.PropertyBool("True if this is an all-day event without specific times"),
		"timezone": map[string]any{
			"type":        "string",
			"description": "Timezone if explicitly mentioned (e.g., 'PST', 'America/New_York'). Optional.",
		},
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0"),
		"raw_text": agent.PropertyString("The original text that was parsed for date/time"),
		"reasoning": agent.PropertyString("Brief explanation of how the date/time was interpreted"),
	}, []string{"has_datetime", "confidence", "reasoning"}),
}

// DateTimeExtraction represents the result of datetime extraction
type DateTimeExtraction struct {
	HasDateTime bool    `json:"has_datetime"`
	StartTime   string  `json:"start_time,omitempty"`
	EndTime     string  `json:"end_time,omitempty"`
	IsAllDay    bool    `json:"is_all_day,omitempty"`
	Timezone    string  `json:"timezone,omitempty"`
	Confidence  float64 `json:"confidence"`
	RawText     string  `json:"raw_text,omitempty"`
	Reasoning   string  `json:"reasoning"`
}

// HandleExtractDateTime processes the extract_datetime tool call
// Note: This handler is called after Claude extracts datetime - it validates and formats the result
func HandleExtractDateTime(_ context.Context, input map[string]any) (string, error) {
	// Parse the input from Claude
	extraction := DateTimeExtraction{}

	if v, ok := input["has_datetime"].(bool); ok {
		extraction.HasDateTime = v
	}
	if v, ok := input["start_time"].(string); ok {
		extraction.StartTime = v
	}
	if v, ok := input["end_time"].(string); ok {
		extraction.EndTime = v
	}
	if v, ok := input["is_all_day"].(bool); ok {
		extraction.IsAllDay = v
	}
	if v, ok := input["timezone"].(string); ok {
		extraction.Timezone = v
	}
	if v, ok := input["confidence"].(float64); ok {
		extraction.Confidence = v
	}
	if v, ok := input["raw_text"].(string); ok {
		extraction.RawText = v
	}
	if v, ok := input["reasoning"].(string); ok {
		extraction.Reasoning = v
	}

	// Validate
	if extraction.HasDateTime && extraction.StartTime == "" {
		return "", fmt.Errorf("start_time is required when has_datetime is true")
	}

	// Return as JSON for the agent to use
	result, err := json.Marshal(extraction)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}
