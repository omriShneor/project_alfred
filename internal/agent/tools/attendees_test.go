package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleExtractAttendees_ValidExtractionWithAttendees(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected AttendeesExtraction
	}{
		{
			name: "single attendee with email",
			input: map[string]any{
				"has_attendees": true,
				"attendees": []any{
					map[string]any{
						"name":  "John Doe",
						"email": "john@example.com",
						"role":  "required",
					},
				},
				"confidence": 0.95,
				"reasoning":  "One person explicitly mentioned with email",
			},
			expected: AttendeesExtraction{
				HasAttendees: true,
				Attendees: []Attendee{
					{
						Name:  "John Doe",
						Email: "john@example.com",
						Role:  "required",
					},
				},
				Confidence: 0.95,
				Reasoning:  "One person explicitly mentioned with email",
			},
		},
		{
			name: "multiple attendees with different roles",
			input: map[string]any{
				"has_attendees": true,
				"attendees": []any{
					map[string]any{
						"name":  "Sarah Smith",
						"email": "sarah@example.com",
						"role":  "organizer",
					},
					map[string]any{
						"name": "Bob Johnson",
						"role": "required",
					},
					map[string]any{
						"name":  "Alice Williams",
						"phone": "+1234567890",
						"role":  "optional",
					},
				},
				"confidence": 0.9,
				"reasoning":  "Three people identified: organizer, required, and optional attendees",
			},
			expected: AttendeesExtraction{
				HasAttendees: true,
				Attendees: []Attendee{
					{
						Name:  "Sarah Smith",
						Email: "sarah@example.com",
						Role:  "organizer",
					},
					{
						Name: "Bob Johnson",
						Role: "required",
					},
					{
						Name:  "Alice Williams",
						Phone: "+1234567890",
						Role:  "optional",
					},
				},
				Confidence: 0.9,
				Reasoning:  "Three people identified: organizer, required, and optional attendees",
			},
		},
		{
			name: "attendee with phone number",
			input: map[string]any{
				"has_attendees": true,
				"attendees": []any{
					map[string]any{
						"name":  "Mike Brown",
						"phone": "+1-555-123-4567",
						"role":  "required",
					},
				},
				"confidence": 0.85,
				"reasoning":  "Person mentioned with phone contact",
			},
			expected: AttendeesExtraction{
				HasAttendees: true,
				Attendees: []Attendee{
					{
						Name:  "Mike Brown",
						Phone: "+1-555-123-4567",
						Role:  "required",
					},
				},
				Confidence: 0.85,
				Reasoning:  "Person mentioned with phone contact",
			},
		},
		{
			name: "attendee with both email and phone",
			input: map[string]any{
				"has_attendees": true,
				"attendees": []any{
					map[string]any{
						"name":  "Emily Davis",
						"email": "emily@example.com",
						"phone": "+1234567890",
						"role":  "organizer",
					},
				},
				"confidence": 1.0,
				"reasoning":  "Complete contact information provided",
			},
			expected: AttendeesExtraction{
				HasAttendees: true,
				Attendees: []Attendee{
					{
						Name:  "Emily Davis",
						Email: "emily@example.com",
						Phone: "+1234567890",
						Role:  "organizer",
					},
				},
				Confidence: 1.0,
				Reasoning:  "Complete contact information provided",
			},
		},
		{
			name: "minimal attendee with name and role only",
			input: map[string]any{
				"has_attendees": true,
				"attendees": []any{
					map[string]any{
						"name": "Chris Lee",
						"role": "required",
					},
				},
				"confidence": 0.8,
				"reasoning":  "Name mentioned without contact details",
			},
			expected: AttendeesExtraction{
				HasAttendees: true,
				Attendees: []Attendee{
					{
						Name: "Chris Lee",
						Role: "required",
					},
				},
				Confidence: 0.8,
				Reasoning:  "Name mentioned without contact details",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleExtractAttendees(context.Background(), tt.input)
			require.NoError(t, err)

			var extraction AttendeesExtraction
			err = json.Unmarshal([]byte(result), &extraction)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.HasAttendees, extraction.HasAttendees)
			assert.Equal(t, tt.expected.Confidence, extraction.Confidence)
			assert.Equal(t, tt.expected.Reasoning, extraction.Reasoning)
			require.Len(t, extraction.Attendees, len(tt.expected.Attendees))

			for i, expectedAttendee := range tt.expected.Attendees {
				assert.Equal(t, expectedAttendee.Name, extraction.Attendees[i].Name)
				assert.Equal(t, expectedAttendee.Email, extraction.Attendees[i].Email)
				assert.Equal(t, expectedAttendee.Phone, extraction.Attendees[i].Phone)
				assert.Equal(t, expectedAttendee.Role, extraction.Attendees[i].Role)
			}
		})
	}
}

func TestHandleExtractAttendees_ValidExtractionWithNoAttendees(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected AttendeesExtraction
	}{
		{
			name: "no attendees mentioned",
			input: map[string]any{
				"has_attendees": false,
				"confidence":    0.95,
				"reasoning":     "No other people mentioned besides the user",
			},
			expected: AttendeesExtraction{
				HasAttendees: false,
				Confidence:   0.95,
				Reasoning:    "No other people mentioned besides the user",
			},
		},
		{
			name: "solo event",
			input: map[string]any{
				"has_attendees": false,
				"confidence":    1.0,
				"reasoning":     "Personal reminder, no attendees",
			},
			expected: AttendeesExtraction{
				HasAttendees: false,
				Confidence:   1.0,
				Reasoning:    "Personal reminder, no attendees",
			},
		},
		{
			name: "vague reference to people",
			input: map[string]any{
				"has_attendees": false,
				"confidence":    0.6,
				"reasoning":     "Reference to 'everyone' is too vague to extract specific attendees",
			},
			expected: AttendeesExtraction{
				HasAttendees: false,
				Confidence:   0.6,
				Reasoning:    "Reference to 'everyone' is too vague to extract specific attendees",
			},
		},
		{
			name: "minimal no attendees",
			input: map[string]any{
				"has_attendees": false,
				"confidence":    1.0,
				"reasoning":     "No attendees",
			},
			expected: AttendeesExtraction{
				HasAttendees: false,
				Confidence:   1.0,
				Reasoning:    "No attendees",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleExtractAttendees(context.Background(), tt.input)
			require.NoError(t, err)

			var extraction AttendeesExtraction
			err = json.Unmarshal([]byte(result), &extraction)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.HasAttendees, extraction.HasAttendees)
			assert.Equal(t, tt.expected.Confidence, extraction.Confidence)
			assert.Equal(t, tt.expected.Reasoning, extraction.Reasoning)
			assert.Empty(t, extraction.Attendees)
		})
	}
}

func TestHandleExtractAttendees_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]any
		expectedErr string
	}{
		{
			name: "has_attendees true but empty array",
			input: map[string]any{
				"has_attendees": true,
				"attendees":     []any{},
				"confidence":    0.8,
				"reasoning":     "No attendees in array",
			},
			expectedErr: "attendees array is required when has_attendees is true",
		},
		{
			name: "has_attendees true but no attendees field",
			input: map[string]any{
				"has_attendees": true,
				"confidence":    0.8,
				"reasoning":     "Missing attendees field",
			},
			expectedErr: "attendees array is required when has_attendees is true",
		},
		{
			name: "has_attendees true but nil attendees",
			input: map[string]any{
				"has_attendees": true,
				"attendees":     nil,
				"confidence":    0.8,
				"reasoning":     "Nil attendees",
			},
			expectedErr: "attendees array is required when has_attendees is true",
		},
		{
			name: "has_attendees true with attendees missing names",
			input: map[string]any{
				"has_attendees": true,
				"attendees": []any{
					map[string]any{
						"email": "john@example.com",
						"role":  "required",
						// name is missing
					},
				},
				"confidence": 0.8,
				"reasoning":  "Attendee without name",
			},
			expectedErr: "attendees array is required when has_attendees is true",
		},
		{
			name: "has_attendees true with attendees having empty names",
			input: map[string]any{
				"has_attendees": true,
				"attendees": []any{
					map[string]any{
						"name":  "", // empty name
						"email": "john@example.com",
						"role":  "required",
					},
				},
				"confidence": 0.8,
				"reasoning":  "Attendee with empty name",
			},
			expectedErr: "attendees array is required when has_attendees is true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleExtractAttendees(context.Background(), tt.input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Empty(t, result)
		})
	}
}

func TestHandleExtractAttendees_AttendeeArrayParsing(t *testing.T) {
	t.Run("filters out attendees with missing names", func(t *testing.T) {
		input := map[string]any{
			"has_attendees": true,
			"attendees": []any{
				map[string]any{
					"name":  "Valid Person",
					"email": "valid@example.com",
					"role":  "required",
				},
				map[string]any{
					// missing name
					"email": "invalid@example.com",
					"role":  "required",
				},
				map[string]any{
					"name": "", // empty name
					"role": "optional",
				},
			},
			"confidence": 0.8,
			"reasoning":  "Mixed valid and invalid attendees",
		}

		result, err := HandleExtractAttendees(context.Background(), input)
		require.NoError(t, err)

		var extraction AttendeesExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)

		assert.True(t, extraction.HasAttendees)
		require.Len(t, extraction.Attendees, 1)
		assert.Equal(t, "Valid Person", extraction.Attendees[0].Name)
	})

	t.Run("handles malformed attendee objects gracefully", func(t *testing.T) {
		input := map[string]any{
			"has_attendees": true,
			"attendees": []any{
				map[string]any{
					"name": "Good Attendee",
					"role": "required",
				},
				"invalid string entry", // not a map
				123,                    // not a map
			},
			"confidence": 0.7,
			"reasoning":  "Some invalid entries",
		}

		result, err := HandleExtractAttendees(context.Background(), input)
		require.NoError(t, err)

		var extraction AttendeesExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)

		assert.True(t, extraction.HasAttendees)
		require.Len(t, extraction.Attendees, 1)
		assert.Equal(t, "Good Attendee", extraction.Attendees[0].Name)
	})
}

func TestHandleExtractAttendees_TypeConversion(t *testing.T) {
	t.Run("handles wrong type for boolean", func(t *testing.T) {
		input := map[string]any{
			"has_attendees": "true", // string instead of bool
			"attendees": []any{
				map[string]any{
					"name": "John Doe",
					"role": "required",
				},
			},
			"confidence": 0.9,
			"reasoning":  "Valid",
		}

		result, err := HandleExtractAttendees(context.Background(), input)
		require.NoError(t, err) // No error because has_attendees defaults to false

		var extraction AttendeesExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)
		assert.False(t, extraction.HasAttendees) // defaults to false on bad type
	})

	t.Run("handles wrong type for number", func(t *testing.T) {
		input := map[string]any{
			"has_attendees": false,
			"confidence":    "0.9", // string instead of float64
			"reasoning":     "Valid",
		}

		result, err := HandleExtractAttendees(context.Background(), input)
		require.NoError(t, err)

		var extraction AttendeesExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)

		assert.Equal(t, 0.0, extraction.Confidence) // defaults to 0.0 on bad type
	})

	t.Run("handles attendees field that is not an array", func(t *testing.T) {
		input := map[string]any{
			"has_attendees": true,
			"attendees":     "not an array", // wrong type
			"confidence":    0.9,
			"reasoning":     "Invalid type",
		}

		_, err := HandleExtractAttendees(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "attendees array is required when has_attendees is true")
	})
}

func TestHandleExtractAttendees_EmptyInput(t *testing.T) {
	input := map[string]any{}

	result, err := HandleExtractAttendees(context.Background(), input)
	require.NoError(t, err) // Should not error with empty input, just defaults

	var extraction AttendeesExtraction
	err = json.Unmarshal([]byte(result), &extraction)
	require.NoError(t, err)

	assert.False(t, extraction.HasAttendees)
	assert.Empty(t, extraction.Attendees)
	assert.Equal(t, 0.0, extraction.Confidence)
	assert.Empty(t, extraction.Reasoning)
}
