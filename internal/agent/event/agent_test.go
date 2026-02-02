package event

import (
	"testing"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAgentOutput_EmptyToolCalls(t *testing.T) {
	output := &agent.AgentOutput{
		ToolCalls: []agent.ToolCall{},
	}

	result, err := parseAgentOutput(output)

	require.NoError(t, err)
	assert.False(t, result.HasEvent)
	assert.Equal(t, "none", result.Action)
	assert.Equal(t, "No tools were called", result.Reasoning)
	assert.Equal(t, float64(0), result.Confidence)
	assert.Nil(t, result.Event)
}

func TestParseAgentOutput_CreateCalendarEvent(t *testing.T) {
	tests := []struct {
		name           string
		output         *agent.AgentOutput
		expectedEvent  *agent.EventData
		expectedReason string
		expectedConf   float64
	}{
		{
			name: "create with all fields",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "create_calendar_event",
						Output: `{
							"status": "success",
							"action": "create",
							"event": {
								"title": "Team Meeting",
								"description": "Weekly sync",
								"start_time": "2024-01-15T14:00:00Z",
								"end_time": "2024-01-15T15:00:00Z",
								"location": "Conference Room A",
								"confidence": 0.95,
								"reasoning": "Clear scheduling intent"
							}
						}`,
					},
				},
			},
			expectedEvent: &agent.EventData{
				Title:       "Team Meeting",
				Description: "Weekly sync",
				StartTime:   "2024-01-15T14:00:00Z",
				EndTime:     "2024-01-15T15:00:00Z",
				Location:    "Conference Room A",
			},
			expectedReason: "Clear scheduling intent",
			expectedConf:   0.95,
		},
		{
			name: "create with minimal fields",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "create_calendar_event",
						Output: `{
							"status": "success",
							"action": "create",
							"event": {
								"title": "Quick Call",
								"start_time": "2024-01-15T10:00:00Z",
								"confidence": 0.85,
								"reasoning": "User wants to schedule"
							}
						}`,
					},
				},
			},
			expectedEvent: &agent.EventData{
				Title:     "Quick Call",
				StartTime: "2024-01-15T10:00:00Z",
			},
			expectedReason: "User wants to schedule",
			expectedConf:   0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAgentOutput(tt.output)

			require.NoError(t, err)
			assert.True(t, result.HasEvent)
			assert.Equal(t, "create", result.Action)
			assert.Equal(t, tt.expectedConf, result.Confidence)
			assert.Equal(t, tt.expectedReason, result.Reasoning)

			require.NotNil(t, result.Event)
			assert.Equal(t, tt.expectedEvent.Title, result.Event.Title)
			assert.Equal(t, tt.expectedEvent.Description, result.Event.Description)
			assert.Equal(t, tt.expectedEvent.StartTime, result.Event.StartTime)
			assert.Equal(t, tt.expectedEvent.EndTime, result.Event.EndTime)
			assert.Equal(t, tt.expectedEvent.Location, result.Event.Location)
		})
	}
}

func TestParseAgentOutput_UpdateCalendarEvent(t *testing.T) {
	tests := []struct {
		name           string
		output         *agent.AgentOutput
		expectedEvent  *agent.EventData
		expectedReason string
		expectedConf   float64
	}{
		{
			name: "update with alfred_event_id",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "update_calendar_event",
						Output: `{
							"status": "success",
							"action": "update",
							"event": {
								"alfred_event_id": 123,
								"title": "Updated Meeting",
								"start_time": "2024-01-15T15:00:00Z",
								"confidence": 0.90,
								"reasoning": "Time changed to 3pm"
							}
						}`,
					},
				},
			},
			expectedEvent: &agent.EventData{
				AlfredEventRef: 123,
				Title:          "Updated Meeting",
				StartTime:      "2024-01-15T15:00:00Z",
			},
			expectedReason: "Time changed to 3pm",
			expectedConf:   0.90,
		},
		{
			name: "update with google_event_id",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "update_calendar_event",
						Output: `{
							"status": "success",
							"action": "update",
							"event": {
								"google_event_id": "google-event-456",
								"location": "New Room",
								"confidence": 0.85,
								"reasoning": "Location changed"
							}
						}`,
					},
				},
			},
			expectedEvent: &agent.EventData{
				UpdateRef: "google-event-456",
				Location:  "New Room",
			},
			expectedReason: "Location changed",
			expectedConf:   0.85,
		},
		{
			name: "update with all optional fields",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "update_calendar_event",
						Output: `{
							"status": "success",
							"action": "update",
							"event": {
								"alfred_event_id": 789,
								"title": "Complete Update",
								"description": "New description",
								"start_time": "2024-01-20T09:00:00Z",
								"end_time": "2024-01-20T10:30:00Z",
								"location": "Virtual - Zoom",
								"confidence": 0.95,
								"reasoning": "Complete event rewrite"
							}
						}`,
					},
				},
			},
			expectedEvent: &agent.EventData{
				AlfredEventRef: 789,
				Title:          "Complete Update",
				Description:    "New description",
				StartTime:      "2024-01-20T09:00:00Z",
				EndTime:        "2024-01-20T10:30:00Z",
				Location:       "Virtual - Zoom",
			},
			expectedReason: "Complete event rewrite",
			expectedConf:   0.95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAgentOutput(tt.output)

			require.NoError(t, err)
			assert.True(t, result.HasEvent)
			assert.Equal(t, "update", result.Action)
			assert.Equal(t, tt.expectedConf, result.Confidence)
			assert.Equal(t, tt.expectedReason, result.Reasoning)

			require.NotNil(t, result.Event)
			assert.Equal(t, tt.expectedEvent.AlfredEventRef, result.Event.AlfredEventRef)
			assert.Equal(t, tt.expectedEvent.UpdateRef, result.Event.UpdateRef)
			assert.Equal(t, tt.expectedEvent.Title, result.Event.Title)
			assert.Equal(t, tt.expectedEvent.Description, result.Event.Description)
			assert.Equal(t, tt.expectedEvent.StartTime, result.Event.StartTime)
			assert.Equal(t, tt.expectedEvent.EndTime, result.Event.EndTime)
			assert.Equal(t, tt.expectedEvent.Location, result.Event.Location)
		})
	}
}

func TestParseAgentOutput_DeleteCalendarEvent(t *testing.T) {
	tests := []struct {
		name           string
		output         *agent.AgentOutput
		expectedEvent  *agent.EventData
		expectedReason string
		expectedConf   float64
	}{
		{
			name: "delete with alfred_event_id",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "delete_calendar_event",
						Output: `{
							"status": "success",
							"action": "delete",
							"event": {
								"alfred_event_id": 123,
								"reason": "Meeting cancelled",
								"confidence": 0.95
							}
						}`,
					},
				},
			},
			expectedEvent: &agent.EventData{
				AlfredEventRef: 123,
			},
			expectedReason: "Meeting cancelled",
			expectedConf:   0.95,
		},
		{
			name: "delete with google_event_id",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "delete_calendar_event",
						Output: `{
							"status": "success",
							"action": "delete",
							"event": {
								"google_event_id": "google-event-456",
								"reason": "Conflict detected",
								"confidence": 0.88
							}
						}`,
					},
				},
			},
			expectedEvent: &agent.EventData{
				UpdateRef: "google-event-456",
			},
			expectedReason: "Conflict detected",
			expectedConf:   0.88,
		},
		{
			name: "delete with both IDs",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "delete_calendar_event",
						Output: `{
							"status": "success",
							"action": "delete",
							"event": {
								"alfred_event_id": 789,
								"google_event_id": "google-event-789",
								"reason": "Event no longer needed",
								"confidence": 0.92
							}
						}`,
					},
				},
			},
			expectedEvent: &agent.EventData{
				AlfredEventRef: 789,
				UpdateRef:      "google-event-789",
			},
			expectedReason: "Event no longer needed",
			expectedConf:   0.92,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAgentOutput(tt.output)

			require.NoError(t, err)
			assert.True(t, result.HasEvent)
			assert.Equal(t, "delete", result.Action)
			assert.Equal(t, tt.expectedConf, result.Confidence)
			assert.Equal(t, tt.expectedReason, result.Reasoning)

			require.NotNil(t, result.Event)
			assert.Equal(t, tt.expectedEvent.AlfredEventRef, result.Event.AlfredEventRef)
			assert.Equal(t, tt.expectedEvent.UpdateRef, result.Event.UpdateRef)
		})
	}
}

func TestParseAgentOutput_NoCalendarAction(t *testing.T) {
	tests := []struct {
		name           string
		output         *agent.AgentOutput
		expectedReason string
		expectedConf   float64
	}{
		{
			name: "no action with reasoning",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "no_calendar_action",
						Output: `{
							"status": "success",
							"action": "none",
							"reasoning": "Just casual chat",
							"confidence": 0.95
						}`,
					},
				},
			},
			expectedReason: "Just casual chat",
			expectedConf:   0.95,
		},
		{
			name: "no action discussing past event",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "no_calendar_action",
						Output: `{
							"status": "success",
							"action": "none",
							"reasoning": "Discussing a past event",
							"confidence": 0.90
						}`,
					},
				},
			},
			expectedReason: "Discussing a past event",
			expectedConf:   0.90,
		},
		{
			name: "no action vague mention",
			output: &agent.AgentOutput{
				ToolCalls: []agent.ToolCall{
					{
						Name: "no_calendar_action",
						Output: `{
							"status": "success",
							"action": "none",
							"reasoning": "Vague time mention without specifics",
							"confidence": 0.75
						}`,
					},
				},
			},
			expectedReason: "Vague time mention without specifics",
			expectedConf:   0.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAgentOutput(tt.output)

			require.NoError(t, err)
			assert.False(t, result.HasEvent)
			assert.Equal(t, "none", result.Action)
			assert.Equal(t, tt.expectedConf, result.Confidence)
			assert.Equal(t, tt.expectedReason, result.Reasoning)
			assert.Nil(t, result.Event)
		})
	}
}

func TestParseAgentOutput_OnlyExtractionTools(t *testing.T) {
	output := &agent.AgentOutput{
		ToolCalls: []agent.ToolCall{
			{
				Name:   "extract_datetime",
				Output: `{"start_time": "2024-01-15T14:00:00Z", "confidence": 0.9}`,
			},
			{
				Name:   "extract_location",
				Output: `{"name": "Conference Room", "confidence": 0.85}`,
			},
			{
				Name:   "extract_attendees",
				Output: `{"attendees": [{"name": "John", "role": "required"}]}`,
			},
		},
	}

	result, err := parseAgentOutput(output)

	require.NoError(t, err)
	assert.False(t, result.HasEvent)
	assert.Equal(t, "none", result.Action)
	assert.Equal(t, "No action tool was called", result.Reasoning)
	assert.Equal(t, float64(0), result.Confidence)
	assert.Nil(t, result.Event)
}

func TestParseAgentOutput_ExtractionFollowedByAction(t *testing.T) {
	t.Run("extraction then create", func(t *testing.T) {
		output := &agent.AgentOutput{
			ToolCalls: []agent.ToolCall{
				{
					Name:   "extract_datetime",
					Output: `{"start_time": "2024-01-15T14:00:00Z"}`,
				},
				{
					Name:   "extract_location",
					Output: `{"name": "Room 101"}`,
				},
				{
					Name: "create_calendar_event",
					Output: `{
						"status": "success",
						"action": "create",
						"event": {
							"title": "Meeting",
							"start_time": "2024-01-15T14:00:00Z",
							"location": "Room 101",
							"confidence": 0.9,
							"reasoning": "Extracted and created"
						}
					}`,
				},
			},
		}

		result, err := parseAgentOutput(output)

		require.NoError(t, err)
		assert.True(t, result.HasEvent)
		assert.Equal(t, "create", result.Action)
		assert.Equal(t, "Extracted and created", result.Reasoning)
		assert.NotNil(t, result.Event)
		assert.Equal(t, "Meeting", result.Event.Title)
	})

	t.Run("extraction then no action", func(t *testing.T) {
		output := &agent.AgentOutput{
			ToolCalls: []agent.ToolCall{
				{
					Name:   "extract_datetime",
					Output: `{"start_time": "2024-01-15T14:00:00Z"}`,
				},
				{
					Name: "no_calendar_action",
					Output: `{
						"status": "success",
						"action": "none",
						"reasoning": "Date mentioned but no scheduling intent",
						"confidence": 0.85
					}`,
				},
			},
		}

		result, err := parseAgentOutput(output)

		require.NoError(t, err)
		assert.False(t, result.HasEvent)
		assert.Equal(t, "none", result.Action)
		assert.Equal(t, "Date mentioned but no scheduling intent", result.Reasoning)
	})
}

func TestParseAgentOutput_InvalidJSON(t *testing.T) {
	output := &agent.AgentOutput{
		ToolCalls: []agent.ToolCall{
			{
				Name:   "create_calendar_event",
				Output: `{invalid json}`,
			},
		},
	}

	result, err := parseAgentOutput(output)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse action result")
}

func TestParseAgentOutput_MissingActionField(t *testing.T) {
	output := &agent.AgentOutput{
		ToolCalls: []agent.ToolCall{
			{
				Name: "create_calendar_event",
				Output: `{
					"status": "success",
					"event": {
						"title": "Meeting",
						"start_time": "2024-01-15T14:00:00Z"
					}
				}`,
			},
		},
	}

	result, err := parseAgentOutput(output)

	require.NoError(t, err)
	// When action field is missing, action defaults to empty string ""
	// HasEvent is true because "" != "none"
	assert.True(t, result.HasEvent)
	assert.Equal(t, "", result.Action)
}

func TestParseEventData_AllFields(t *testing.T) {
	data := map[string]any{
		"title":            "Team Meeting",
		"description":      "Weekly sync meeting",
		"start_time":       "2024-01-15T14:00:00Z",
		"end_time":         "2024-01-15T15:00:00Z",
		"location":         "Conference Room A",
		"alfred_event_id":  float64(123),
		"google_event_id":  "google-event-456",
	}

	event := parseEventData(data)

	assert.Equal(t, "Team Meeting", event.Title)
	assert.Equal(t, "Weekly sync meeting", event.Description)
	assert.Equal(t, "2024-01-15T14:00:00Z", event.StartTime)
	assert.Equal(t, "2024-01-15T15:00:00Z", event.EndTime)
	assert.Equal(t, "Conference Room A", event.Location)
	assert.Equal(t, int64(123), event.AlfredEventRef)
	assert.Equal(t, "google-event-456", event.UpdateRef)
}

func TestParseEventData_MinimalFields(t *testing.T) {
	data := map[string]any{
		"title":      "Quick Meeting",
		"start_time": "2024-01-15T10:00:00Z",
	}

	event := parseEventData(data)

	assert.Equal(t, "Quick Meeting", event.Title)
	assert.Equal(t, "2024-01-15T10:00:00Z", event.StartTime)
	assert.Empty(t, event.Description)
	assert.Empty(t, event.EndTime)
	assert.Empty(t, event.Location)
	assert.Equal(t, int64(0), event.AlfredEventRef)
	assert.Empty(t, event.UpdateRef)
}

func TestParseEventData_EmptyMap(t *testing.T) {
	data := map[string]any{}

	event := parseEventData(data)

	assert.NotNil(t, event)
	assert.Empty(t, event.Title)
	assert.Empty(t, event.Description)
	assert.Empty(t, event.StartTime)
	assert.Empty(t, event.EndTime)
	assert.Empty(t, event.Location)
	assert.Equal(t, int64(0), event.AlfredEventRef)
	assert.Empty(t, event.UpdateRef)
}

func TestParseEventData_WrongTypes(t *testing.T) {
	data := map[string]any{
		"title":            123,              // Should be string
		"description":      true,             // Should be string
		"start_time":       []string{"time"}, // Should be string
		"alfred_event_id":  "not-a-number",   // Should be float64
		"google_event_id":  456,              // Should be string
	}

	event := parseEventData(data)

	// Type assertions fail, fields remain zero/empty
	assert.Empty(t, event.Title)
	assert.Empty(t, event.Description)
	assert.Empty(t, event.StartTime)
	assert.Equal(t, int64(0), event.AlfredEventRef)
	assert.Empty(t, event.UpdateRef)
}

func TestParseEventData_PartialFields(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		checkFn  func(*testing.T, *agent.EventData)
	}{
		{
			name: "only title and start time",
			data: map[string]any{
				"title":      "Meeting",
				"start_time": "2024-01-15T14:00:00Z",
			},
			checkFn: func(t *testing.T, event *agent.EventData) {
				assert.Equal(t, "Meeting", event.Title)
				assert.Equal(t, "2024-01-15T14:00:00Z", event.StartTime)
				assert.Empty(t, event.Description)
				assert.Empty(t, event.EndTime)
			},
		},
		{
			name: "only alfred_event_id",
			data: map[string]any{
				"alfred_event_id": float64(999),
			},
			checkFn: func(t *testing.T, event *agent.EventData) {
				assert.Equal(t, int64(999), event.AlfredEventRef)
				assert.Empty(t, event.Title)
				assert.Empty(t, event.UpdateRef)
			},
		},
		{
			name: "only google_event_id",
			data: map[string]any{
				"google_event_id": "google-123",
			},
			checkFn: func(t *testing.T, event *agent.EventData) {
				assert.Equal(t, "google-123", event.UpdateRef)
				assert.Empty(t, event.Title)
				assert.Equal(t, int64(0), event.AlfredEventRef)
			},
		},
		{
			name: "description and location only",
			data: map[string]any{
				"description": "Details here",
				"location":    "Room 202",
			},
			checkFn: func(t *testing.T, event *agent.EventData) {
				assert.Equal(t, "Details here", event.Description)
				assert.Equal(t, "Room 202", event.Location)
				assert.Empty(t, event.Title)
				assert.Empty(t, event.StartTime)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := parseEventData(tt.data)
			require.NotNil(t, event)
			tt.checkFn(t, event)
		})
	}
}

func TestParseEventData_FloatConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int64
	}{
		{
			name:     "small positive float",
			input:    float64(123),
			expected: 123,
		},
		{
			name:     "large float",
			input:    float64(999999),
			expected: 999999,
		},
		{
			name:     "zero",
			input:    float64(0),
			expected: 0,
		},
		{
			name:     "float with decimal (truncated)",
			input:    float64(123.456),
			expected: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]any{
				"alfred_event_id": tt.input,
			}

			event := parseEventData(data)
			assert.Equal(t, tt.expected, event.AlfredEventRef)
		})
	}
}

func TestParseAgentOutput_MultipleActionTools(t *testing.T) {
	// If multiple action tools are called, the last one should take precedence
	output := &agent.AgentOutput{
		ToolCalls: []agent.ToolCall{
			{
				Name: "create_calendar_event",
				Output: `{
					"status": "success",
					"action": "create",
					"event": {
						"title": "First Event",
						"start_time": "2024-01-15T14:00:00Z",
						"confidence": 0.8,
						"reasoning": "First attempt"
					}
				}`,
			},
			{
				Name: "no_calendar_action",
				Output: `{
					"status": "success",
					"action": "none",
					"reasoning": "Actually, no action needed",
					"confidence": 0.9
				}`,
			},
		},
	}

	result, err := parseAgentOutput(output)

	require.NoError(t, err)
	// The last action tool (no_calendar_action) should be used
	assert.False(t, result.HasEvent)
	assert.Equal(t, "none", result.Action)
	assert.Equal(t, "Actually, no action needed", result.Reasoning)
	assert.Equal(t, 0.9, result.Confidence)
}
