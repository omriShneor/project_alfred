package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
)

// ExtractLocationTool extracts location information from text
var ExtractLocationTool = agent.Tool{
	Name: "extract_location",
	Description: `Extracts location information from text for calendar events.
Handles physical addresses ("123 Main St"), venue names ("Starbucks on 5th Ave"),
virtual meeting links (Zoom, Google Meet, Teams URLs), and contextual references
("at the office", "at Sarah's place", "usual spot"). Returns structured location
with type classification. If no location is mentioned, return has_location: false.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"has_location": agent.PropertyBool("Whether the text contains location information"),
		"name": map[string]any{
			"type":        "string",
			"description": "Location name or description (e.g., 'Starbucks', 'Conference Room A', 'Zoom Meeting')",
		},
		"address": map[string]any{
			"type":        "string",
			"description": "Full street address if available. Optional.",
		},
		"type": agent.PropertyEnum("Type of location", []string{"physical", "virtual", "unknown"}),
		"url": map[string]any{
			"type":        "string",
			"description": "Meeting URL for virtual locations (Zoom, Meet, Teams links). Optional.",
		},
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0"),
		"raw_text": agent.PropertyString("The original text that was parsed for location"),
		"reasoning": agent.PropertyString("Brief explanation of how the location was interpreted"),
	}, []string{"has_location", "confidence", "reasoning"}),
}

// LocationExtraction represents the result of location extraction
type LocationExtraction struct {
	HasLocation bool    `json:"has_location"`
	Name        string  `json:"name,omitempty"`
	Address     string  `json:"address,omitempty"`
	Type        string  `json:"type,omitempty"` // "physical", "virtual", "unknown"
	URL         string  `json:"url,omitempty"`
	Confidence  float64 `json:"confidence"`
	RawText     string  `json:"raw_text,omitempty"`
	Reasoning   string  `json:"reasoning"`
}

// HandleExtractLocation processes the extract_location tool call
func HandleExtractLocation(_ context.Context, input map[string]any) (string, error) {
	extraction := LocationExtraction{}

	if v, ok := input["has_location"].(bool); ok {
		extraction.HasLocation = v
	}
	if v, ok := input["name"].(string); ok {
		extraction.Name = v
	}
	if v, ok := input["address"].(string); ok {
		extraction.Address = v
	}
	if v, ok := input["type"].(string); ok {
		extraction.Type = v
	}
	if v, ok := input["url"].(string); ok {
		extraction.URL = v
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
	if extraction.HasLocation && extraction.Name == "" {
		return "", fmt.Errorf("name is required when has_location is true")
	}

	result, err := json.Marshal(extraction)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}
