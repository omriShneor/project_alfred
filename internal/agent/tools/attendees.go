package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
)

// ExtractAttendeesTool extracts attendee information from text
var ExtractAttendeesTool = agent.Tool{
	Name: "extract_attendees",
	Description: `Extracts information about people who should be invited to a calendar event.
Identifies names mentioned in the context of scheduling or meetings. Extracts email
addresses or phone numbers when available. Distinguishes between the organizer (person
initiating), required attendees, and optional attendees. Does NOT include the message
recipient (the user) as an attendee - only extract OTHER people mentioned.`,
	InputSchema: agent.BuildJSONSchema("object", map[string]any{
		"has_attendees": agent.PropertyBool("Whether other attendees (besides the user) are mentioned"),
		"attendees": agent.PropertyArray("List of attendees to invite", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":  agent.PropertyString("Person's name"),
				"email": agent.PropertyString("Email address if mentioned. Optional."),
				"phone": agent.PropertyString("Phone number if mentioned. Optional."),
				"role":  agent.PropertyEnum("Role in the event", []string{"organizer", "required", "optional"}),
			},
			"required": []string{"name", "role"},
		}),
		"confidence": agent.PropertyNumber("Confidence score from 0.0 to 1.0"),
		"reasoning": agent.PropertyString("Brief explanation of who was identified and why"),
	}, []string{"has_attendees", "confidence", "reasoning"}),
}

// Attendee represents a single attendee
type Attendee struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
	Role  string `json:"role"` // "organizer", "required", "optional"
}

// AttendeesExtraction represents the result of attendees extraction
type AttendeesExtraction struct {
	HasAttendees bool       `json:"has_attendees"`
	Attendees    []Attendee `json:"attendees,omitempty"`
	Confidence   float64    `json:"confidence"`
	Reasoning    string     `json:"reasoning"`
}

// HandleExtractAttendees processes the extract_attendees tool call
func HandleExtractAttendees(_ context.Context, input map[string]any) (string, error) {
	extraction := AttendeesExtraction{}

	if v, ok := input["has_attendees"].(bool); ok {
		extraction.HasAttendees = v
	}
	if v, ok := input["confidence"].(float64); ok {
		extraction.Confidence = v
	}
	if v, ok := input["reasoning"].(string); ok {
		extraction.Reasoning = v
	}

	// Parse attendees array
	if attendeesRaw, ok := input["attendees"].([]any); ok {
		for _, a := range attendeesRaw {
			if aMap, ok := a.(map[string]any); ok {
				attendee := Attendee{}
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
				if attendee.Name != "" {
					extraction.Attendees = append(extraction.Attendees, attendee)
				}
			}
		}
	}

	// Validate
	if extraction.HasAttendees && len(extraction.Attendees) == 0 {
		return "", fmt.Errorf("attendees array is required when has_attendees is true")
	}

	result, err := json.Marshal(extraction)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}
