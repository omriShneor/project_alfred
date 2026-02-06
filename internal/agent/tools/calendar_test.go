package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCreateCalendarEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]any
		wantErr   bool
		errMsg    string
		checkJSON bool // If true, verify JSON structure
	}{
		{
			name: "valid input with all fields",
			input: map[string]any{
				"title":       "Team Meeting",
				"description": "Weekly sync meeting",
				"start_time":  "2024-01-15T14:00:00Z",
				"end_time":    "2024-01-15T15:00:00Z",
				"location":    "Conference Room A",
				"confidence":  0.95,
				"reasoning":   "User mentioned a scheduled meeting",
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid input with required fields only",
			input: map[string]any{
				"title":      "Quick Call",
				"start_time": "2024-01-15T10:00:00Z",
				"confidence": 0.85,
				"reasoning":  "User wants to schedule a call",
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "missing title",
			input: map[string]any{
				"start_time": "2024-01-15T14:00:00Z",
				"confidence": 0.90,
				"reasoning":  "Event without title",
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "missing start_time",
			input: map[string]any{
				"title":      "Meeting",
				"confidence": 0.90,
				"reasoning":  "Event without start time",
			},
			wantErr: true,
			errMsg:  "start_time is required",
		},
		{
			name: "empty title",
			input: map[string]any{
				"title":      "",
				"start_time": "2024-01-15T14:00:00Z",
				"confidence": 0.90,
				"reasoning":  "Event with empty title",
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "empty start_time",
			input: map[string]any{
				"title":      "Meeting",
				"start_time": "",
				"confidence": 0.90,
				"reasoning":  "Event with empty start time",
			},
			wantErr: true,
			errMsg:  "start_time is required",
		},
		{
			name: "valid with zero confidence",
			input: map[string]any{
				"title":      "Low confidence event",
				"start_time": "2024-01-15T14:00:00Z",
				"confidence": 0.0,
				"reasoning":  "Not sure about this",
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid with high confidence",
			input: map[string]any{
				"title":      "High confidence event",
				"start_time": "2024-01-15T14:00:00Z",
				"confidence": 1.0,
				"reasoning":  "Very clear event",
			},
			wantErr:   false,
			checkJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleCreateCalendarEvent(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, result)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, result)

			if tt.checkJSON {
				// Verify JSON structure
				var parsed map[string]any
				err := json.Unmarshal([]byte(result), &parsed)
				require.NoError(t, err)

				assert.Equal(t, "success", parsed["status"])
				assert.Equal(t, "create", parsed["action"])
				assert.Contains(t, parsed, "event")

				event, ok := parsed["event"].(map[string]any)
				require.True(t, ok, "event should be a map")

				// Verify required fields are present in the result
				assert.Equal(t, tt.input["title"], event["title"])
				assert.Equal(t, tt.input["start_time"], event["start_time"])
				assert.Equal(t, tt.input["confidence"], event["confidence"])
				assert.Equal(t, tt.input["reasoning"], event["reasoning"])

				// Verify optional fields if provided
				if desc, ok := tt.input["description"]; ok {
					assert.Equal(t, desc, event["description"])
				}
				if endTime, ok := tt.input["end_time"]; ok {
					assert.Equal(t, endTime, event["end_time"])
				}
				if loc, ok := tt.input["location"]; ok {
					assert.Equal(t, loc, event["location"])
				}
			}
		})
	}
}

func TestHandleUpdateCalendarEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]any
		wantErr   bool
		errMsg    string
		checkJSON bool
	}{
		{
			name: "valid input with alfred_event_id",
			input: map[string]any{
				"alfred_event_id": float64(123),
				"title":           "Updated Meeting",
				"start_time":      "2024-01-15T15:00:00Z",
				"confidence":      0.90,
				"reasoning":       "Time changed to 3pm",
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid input with google_event_id",
			input: map[string]any{
				"google_event_id": "google-event-123",
				"location":        "New Conference Room",
				"confidence":      0.85,
				"reasoning":       "Location changed",
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid input with both IDs (alfred_event_id takes precedence)",
			input: map[string]any{
				"alfred_event_id": float64(456),
				"google_event_id": "google-event-456",
				"title":           "Updated Title",
				"confidence":      0.92,
				"reasoning":       "Both IDs provided",
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid input updating only time",
			input: map[string]any{
				"alfred_event_id": float64(789),
				"start_time":      "2024-01-16T10:00:00Z",
				"end_time":        "2024-01-16T11:00:00Z",
				"confidence":      0.88,
				"reasoning":       "Rescheduled to next day",
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "error when neither ID provided",
			input: map[string]any{
				"title":      "Updated Meeting",
				"confidence": 0.90,
				"reasoning":  "No event reference",
			},
			wantErr: true,
			errMsg:  "either alfred_event_id or google_event_id is required",
		},
		{
			name: "error with zero alfred_event_id and empty google_event_id",
			input: map[string]any{
				"alfred_event_id": float64(0),
				"google_event_id": "",
				"confidence":      0.90,
				"reasoning":       "Both IDs invalid",
			},
			wantErr: true,
			errMsg:  "either alfred_event_id or google_event_id is required",
		},
		{
			name: "valid with only description change",
			input: map[string]any{
				"google_event_id": "google-event-999",
				"description":     "Updated meeting notes",
				"confidence":      0.75,
				"reasoning":       "Added more details",
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid with all optional fields",
			input: map[string]any{
				"alfred_event_id": float64(111),
				"title":           "Complete Update",
				"description":     "New description",
				"start_time":      "2024-01-20T09:00:00Z",
				"end_time":        "2024-01-20T10:30:00Z",
				"location":        "Virtual - Zoom",
				"confidence":      0.95,
				"reasoning":       "Complete event rewrite",
			},
			wantErr:   false,
			checkJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleUpdateCalendarEvent(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, result)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, result)

			if tt.checkJSON {
				var parsed map[string]any
				err := json.Unmarshal([]byte(result), &parsed)
				require.NoError(t, err)

				assert.Equal(t, "success", parsed["status"])
				assert.Equal(t, "update", parsed["action"])
				assert.Contains(t, parsed, "event")

				event, ok := parsed["event"].(map[string]any)
				require.True(t, ok, "event should be a map")

				// Verify required fields
				assert.Equal(t, tt.input["confidence"], event["confidence"])
				assert.Equal(t, tt.input["reasoning"], event["reasoning"])

				// Verify event ID fields
				if alfredID, ok := tt.input["alfred_event_id"]; ok && alfredID.(float64) > 0 {
					assert.Equal(t, int64(alfredID.(float64)), int64(event["alfred_event_id"].(float64)))
				}
				if googleID, ok := tt.input["google_event_id"]; ok && googleID.(string) != "" {
					assert.Equal(t, googleID, event["google_event_id"])
				}

				// Verify optional update fields if provided
				if title, ok := tt.input["title"]; ok {
					assert.Equal(t, title, event["title"])
				}
				if desc, ok := tt.input["description"]; ok {
					assert.Equal(t, desc, event["description"])
				}
				if startTime, ok := tt.input["start_time"]; ok {
					assert.Equal(t, startTime, event["start_time"])
				}
				if endTime, ok := tt.input["end_time"]; ok {
					assert.Equal(t, endTime, event["end_time"])
				}
				if loc, ok := tt.input["location"]; ok {
					assert.Equal(t, loc, event["location"])
				}
			}
		})
	}
}

func TestHandleDeleteCalendarEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]any
		wantErr   bool
		errMsg    string
		checkJSON bool
	}{
		{
			name: "valid input with alfred_event_id",
			input: map[string]any{
				"alfred_event_id": float64(123),
				"reason":          "Meeting cancelled by organizer",
				"confidence":      0.95,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid input with google_event_id",
			input: map[string]any{
				"google_event_id": "google-event-456",
				"reason":          "Conflict with another meeting",
				"confidence":      0.88,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid input with both IDs",
			input: map[string]any{
				"alfred_event_id": float64(789),
				"google_event_id": "google-event-789",
				"reason":          "Event no longer needed",
				"confidence":      0.92,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "error when no event ID provided",
			input: map[string]any{
				"reason":     "Meeting cancelled",
				"confidence": 0.90,
			},
			wantErr: true,
			errMsg:  "either alfred_event_id or google_event_id is required",
		},
		{
			name: "error when both IDs are zero/empty",
			input: map[string]any{
				"alfred_event_id": float64(0),
				"google_event_id": "",
				"reason":          "No valid reference",
				"confidence":      0.90,
			},
			wantErr: true,
			errMsg:  "either alfred_event_id or google_event_id is required",
		},
		{
			name: "error when reason is missing",
			input: map[string]any{
				"alfred_event_id": float64(123),
				"confidence":      0.90,
			},
			wantErr: true,
			errMsg:  "reason is required",
		},
		{
			name: "error when reason is empty",
			input: map[string]any{
				"alfred_event_id": float64(456),
				"reason":          "",
				"confidence":      0.85,
			},
			wantErr: true,
			errMsg:  "reason is required",
		},
		{
			name: "error when reason missing but google_event_id provided",
			input: map[string]any{
				"google_event_id": "google-event-999",
				"confidence":      0.80,
			},
			wantErr: true,
			errMsg:  "reason is required",
		},
		{
			name: "valid with low confidence",
			input: map[string]any{
				"alfred_event_id": float64(111),
				"reason":          "Possibly cancelled",
				"confidence":      0.50,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid with detailed reason",
			input: map[string]any{
				"google_event_id": "google-event-222",
				"reason":          "Project postponed indefinitely, all related meetings cancelled",
				"confidence":      1.0,
			},
			wantErr:   false,
			checkJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleDeleteCalendarEvent(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, result)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, result)

			if tt.checkJSON {
				var parsed map[string]any
				err := json.Unmarshal([]byte(result), &parsed)
				require.NoError(t, err)

				assert.Equal(t, "success", parsed["status"])
				assert.Equal(t, "delete", parsed["action"])
				assert.Contains(t, parsed, "event")

				event, ok := parsed["event"].(map[string]any)
				require.True(t, ok, "event should be a map")

				// Verify required fields
				assert.Equal(t, tt.input["reason"], event["reason"])
				assert.Equal(t, tt.input["confidence"], event["confidence"])

				// Verify event ID fields
				if alfredID, ok := tt.input["alfred_event_id"]; ok && alfredID.(float64) > 0 {
					assert.Equal(t, int64(alfredID.(float64)), int64(event["alfred_event_id"].(float64)))
				}
				if googleID, ok := tt.input["google_event_id"]; ok && googleID.(string) != "" {
					assert.Equal(t, googleID, event["google_event_id"])
				}
			}
		})
	}
}

func TestHandleNoAction(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]any
		wantErr   bool
		errMsg    string
		checkJSON bool
	}{
		{
			name: "valid input with reasoning and confidence",
			input: map[string]any{
				"reasoning":  "Message is just casual chat, no scheduling intent",
				"confidence": 0.95,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid input discussing past event",
			input: map[string]any{
				"reasoning":  "Message discusses a past event that already happened",
				"confidence": 0.90,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid input vague mention",
			input: map[string]any{
				"reasoning":  "Message mentions 'sometime next week' but no specific time",
				"confidence": 0.85,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "error when reasoning is missing",
			input: map[string]any{
				"confidence": 0.90,
			},
			wantErr: true,
			errMsg:  "reasoning is required",
		},
		{
			name: "error when reasoning is empty",
			input: map[string]any{
				"reasoning":  "",
				"confidence": 0.88,
			},
			wantErr: true,
			errMsg:  "reasoning is required",
		},
		{
			name: "valid with zero confidence",
			input: map[string]any{
				"reasoning":  "Unsure if this needs action",
				"confidence": 0.0,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid with high confidence",
			input: map[string]any{
				"reasoning":  "Definitely just general conversation",
				"confidence": 1.0,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid with detailed reasoning",
			input: map[string]any{
				"reasoning":  "While the message mentions 'meeting', it's in the context of discussing what happened in yesterday's meeting, not scheduling a new one. No future scheduling intent detected.",
				"confidence": 0.92,
			},
			wantErr:   false,
			checkJSON: true,
		},
		{
			name: "valid missing confidence (should still work)",
			input: map[string]any{
				"reasoning": "No calendar action needed",
			},
			wantErr:   false,
			checkJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleNoAction(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, result)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, result)

			if tt.checkJSON {
				var parsed map[string]any
				err := json.Unmarshal([]byte(result), &parsed)
				require.NoError(t, err)

				assert.Equal(t, "success", parsed["status"])
				assert.Equal(t, "none", parsed["action"])
				assert.Contains(t, parsed, "reasoning")
				assert.Equal(t, tt.input["reasoning"], parsed["reasoning"])
				if confidence, ok := tt.input["confidence"]; ok {
					assert.Equal(t, confidence, parsed["confidence"])
				} else {
					assert.Equal(t, float64(0), parsed["confidence"])
				}
			}
		})
	}
}

func TestCreateEventInput_Parsing(t *testing.T) {
	// Test that the CreateEventInput struct correctly represents the expected fields
	input := CreateEventInput{
		Title:       "Test Meeting",
		Description: "Test description",
		StartTime:   "2024-01-15T14:00:00Z",
		EndTime:     "2024-01-15T15:00:00Z",
		Location:    "Room 101",
		Confidence:  0.95,
		Reasoning:   "Test reasoning",
	}

	assert.Equal(t, "Test Meeting", input.Title)
	assert.Equal(t, "Test description", input.Description)
	assert.Equal(t, "2024-01-15T14:00:00Z", input.StartTime)
	assert.Equal(t, "2024-01-15T15:00:00Z", input.EndTime)
	assert.Equal(t, "Room 101", input.Location)
	assert.Equal(t, 0.95, input.Confidence)
	assert.Equal(t, "Test reasoning", input.Reasoning)
}

func TestUpdateEventInput_Parsing(t *testing.T) {
	// Test that the UpdateEventInput struct correctly represents the expected fields
	input := UpdateEventInput{
		AlfredEventID: 123,
		GoogleEventID: "google-event-456",
		Title:         "Updated Title",
		Description:   "Updated description",
		StartTime:     "2024-01-16T10:00:00Z",
		EndTime:       "2024-01-16T11:00:00Z",
		Location:      "Room 202",
		Confidence:    0.88,
		Reasoning:     "Update reasoning",
	}

	assert.Equal(t, int64(123), input.AlfredEventID)
	assert.Equal(t, "google-event-456", input.GoogleEventID)
	assert.Equal(t, "Updated Title", input.Title)
	assert.Equal(t, "Updated description", input.Description)
	assert.Equal(t, "2024-01-16T10:00:00Z", input.StartTime)
	assert.Equal(t, "2024-01-16T11:00:00Z", input.EndTime)
	assert.Equal(t, "Room 202", input.Location)
	assert.Equal(t, 0.88, input.Confidence)
	assert.Equal(t, "Update reasoning", input.Reasoning)
}

func TestDeleteEventInput_Parsing(t *testing.T) {
	// Test that the DeleteEventInput struct correctly represents the expected fields
	input := DeleteEventInput{
		AlfredEventID: 789,
		GoogleEventID: "google-event-789",
		Reason:        "Cancelled",
		Confidence:    0.92,
	}

	assert.Equal(t, int64(789), input.AlfredEventID)
	assert.Equal(t, "google-event-789", input.GoogleEventID)
	assert.Equal(t, "Cancelled", input.Reason)
	assert.Equal(t, 0.92, input.Confidence)
}

func TestNoActionInput_Parsing(t *testing.T) {
	// Test that the NoActionInput struct correctly represents the expected fields
	input := NoActionInput{
		Reasoning:  "No action needed",
		Confidence: 0.97,
	}

	assert.Equal(t, "No action needed", input.Reasoning)
	assert.Equal(t, 0.97, input.Confidence)
}

func TestAllCalendarTools(t *testing.T) {
	// Test that AllCalendarTools returns all expected tools
	tools := AllCalendarTools()

	assert.Len(t, tools, 4, "should return 4 calendar tools")

	// Verify all tool names are present
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames["create_calendar_event"], "should include create_calendar_event")
	assert.True(t, toolNames["update_calendar_event"], "should include update_calendar_event")
	assert.True(t, toolNames["delete_calendar_event"], "should include delete_calendar_event")
	assert.True(t, toolNames["no_calendar_action"], "should include no_calendar_action")
}

func TestToolDefinitions(t *testing.T) {
	// Test that tool definitions are properly configured
	t.Run("CreateCalendarEventTool", func(t *testing.T) {
		assert.Equal(t, "create_calendar_event", CreateCalendarEventTool.Name)
		assert.NotEmpty(t, CreateCalendarEventTool.Description)
		assert.NotNil(t, CreateCalendarEventTool.InputSchema)
	})

	t.Run("UpdateCalendarEventTool", func(t *testing.T) {
		assert.Equal(t, "update_calendar_event", UpdateCalendarEventTool.Name)
		assert.NotEmpty(t, UpdateCalendarEventTool.Description)
		assert.NotNil(t, UpdateCalendarEventTool.InputSchema)
	})

	t.Run("DeleteCalendarEventTool", func(t *testing.T) {
		assert.Equal(t, "delete_calendar_event", DeleteCalendarEventTool.Name)
		assert.NotEmpty(t, DeleteCalendarEventTool.Description)
		assert.NotNil(t, DeleteCalendarEventTool.InputSchema)
	})

	t.Run("NoActionTool", func(t *testing.T) {
		assert.Equal(t, "no_calendar_action", NoActionTool.Name)
		assert.NotEmpty(t, NoActionTool.Description)
		assert.NotNil(t, NoActionTool.InputSchema)
	})
}
