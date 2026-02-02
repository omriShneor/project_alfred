package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleExtractDateTime_ValidExtractionWithDateTime(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected DateTimeExtraction
	}{
		{
			name: "complete datetime with end time",
			input: map[string]any{
				"has_datetime": true,
				"start_time":   "2024-02-14T15:00:00",
				"end_time":     "2024-02-14T16:00:00",
				"is_all_day":   false,
				"timezone":     "America/New_York",
				"confidence":   0.95,
				"raw_text":     "February 14th at 3pm to 4pm EST",
				"reasoning":    "Explicit date and time range with timezone",
			},
			expected: DateTimeExtraction{
				HasDateTime: true,
				StartTime:   "2024-02-14T15:00:00",
				EndTime:     "2024-02-14T16:00:00",
				IsAllDay:    false,
				Timezone:    "America/New_York",
				Confidence:  0.95,
				RawText:     "February 14th at 3pm to 4pm EST",
				Reasoning:   "Explicit date and time range with timezone",
			},
		},
		{
			name: "start time only without end time",
			input: map[string]any{
				"has_datetime": true,
				"start_time":   "2024-03-01T09:00:00",
				"is_all_day":   false,
				"confidence":   0.85,
				"raw_text":     "tomorrow at 9am",
				"reasoning":    "Relative date converted to explicit time",
			},
			expected: DateTimeExtraction{
				HasDateTime: true,
				StartTime:   "2024-03-01T09:00:00",
				IsAllDay:    false,
				Confidence:  0.85,
				RawText:     "tomorrow at 9am",
				Reasoning:   "Relative date converted to explicit time",
			},
		},
		{
			name: "all day event",
			input: map[string]any{
				"has_datetime": true,
				"start_time":   "2024-12-25T00:00:00",
				"is_all_day":   true,
				"confidence":   1.0,
				"raw_text":     "Christmas day",
				"reasoning":    "Holiday, all-day event",
			},
			expected: DateTimeExtraction{
				HasDateTime: true,
				StartTime:   "2024-12-25T00:00:00",
				IsAllDay:    true,
				Confidence:  1.0,
				RawText:     "Christmas day",
				Reasoning:   "Holiday, all-day event",
			},
		},
		{
			name: "minimal valid datetime",
			input: map[string]any{
				"has_datetime": true,
				"start_time":   "2024-01-15T14:30:00",
				"confidence":   0.8,
				"reasoning":    "Parsed from text",
			},
			expected: DateTimeExtraction{
				HasDateTime: true,
				StartTime:   "2024-01-15T14:30:00",
				Confidence:  0.8,
				Reasoning:   "Parsed from text",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleExtractDateTime(context.Background(), tt.input)
			require.NoError(t, err)

			var extraction DateTimeExtraction
			err = json.Unmarshal([]byte(result), &extraction)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.HasDateTime, extraction.HasDateTime)
			assert.Equal(t, tt.expected.StartTime, extraction.StartTime)
			assert.Equal(t, tt.expected.EndTime, extraction.EndTime)
			assert.Equal(t, tt.expected.IsAllDay, extraction.IsAllDay)
			assert.Equal(t, tt.expected.Timezone, extraction.Timezone)
			assert.Equal(t, tt.expected.Confidence, extraction.Confidence)
			assert.Equal(t, tt.expected.RawText, extraction.RawText)
			assert.Equal(t, tt.expected.Reasoning, extraction.Reasoning)
		})
	}
}

func TestHandleExtractDateTime_ValidExtractionWithNoDateTime(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected DateTimeExtraction
	}{
		{
			name: "no datetime information",
			input: map[string]any{
				"has_datetime": false,
				"confidence":   0.9,
				"reasoning":    "No temporal information found in text",
				"raw_text":     "Let's catch up sometime",
			},
			expected: DateTimeExtraction{
				HasDateTime: false,
				Confidence:  0.9,
				Reasoning:   "No temporal information found in text",
				RawText:     "Let's catch up sometime",
			},
		},
		{
			name: "vague temporal reference",
			input: map[string]any{
				"has_datetime": false,
				"confidence":   0.5,
				"reasoning":    "Vague reference 'later' without specific time",
			},
			expected: DateTimeExtraction{
				HasDateTime: false,
				Confidence:  0.5,
				Reasoning:   "Vague reference 'later' without specific time",
			},
		},
		{
			name: "minimal no datetime",
			input: map[string]any{
				"has_datetime": false,
				"confidence":   1.0,
				"reasoning":    "Not scheduling related",
			},
			expected: DateTimeExtraction{
				HasDateTime: false,
				Confidence:  1.0,
				Reasoning:   "Not scheduling related",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleExtractDateTime(context.Background(), tt.input)
			require.NoError(t, err)

			var extraction DateTimeExtraction
			err = json.Unmarshal([]byte(result), &extraction)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.HasDateTime, extraction.HasDateTime)
			assert.Empty(t, extraction.StartTime)
			assert.Empty(t, extraction.EndTime)
			assert.Equal(t, tt.expected.Confidence, extraction.Confidence)
			assert.Equal(t, tt.expected.Reasoning, extraction.Reasoning)
		})
	}
}

func TestHandleExtractDateTime_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]any
		expectedErr string
	}{
		{
			name: "has_datetime true but no start_time",
			input: map[string]any{
				"has_datetime": true,
				"confidence":   0.8,
				"reasoning":    "Has datetime",
			},
			expectedErr: "start_time is required when has_datetime is true",
		},
		{
			name: "has_datetime true with empty start_time",
			input: map[string]any{
				"has_datetime": true,
				"start_time":   "",
				"confidence":   0.8,
				"reasoning":    "Has datetime",
			},
			expectedErr: "start_time is required when has_datetime is true",
		},
		{
			name: "has_datetime true with missing start_time field",
			input: map[string]any{
				"has_datetime": true,
				"end_time":     "2024-02-14T16:00:00",
				"confidence":   0.8,
				"reasoning":    "Missing start time",
			},
			expectedErr: "start_time is required when has_datetime is true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleExtractDateTime(context.Background(), tt.input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Empty(t, result)
		})
	}
}

func TestHandleExtractDateTime_TypeConversion(t *testing.T) {
	t.Run("handles missing optional fields gracefully", func(t *testing.T) {
		input := map[string]any{
			"has_datetime": true,
			"start_time":   "2024-02-14T15:00:00",
			"confidence":   0.9,
			"reasoning":    "Valid",
			// end_time, is_all_day, timezone, raw_text omitted
		}

		result, err := HandleExtractDateTime(context.Background(), input)
		require.NoError(t, err)

		var extraction DateTimeExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)

		assert.True(t, extraction.HasDateTime)
		assert.Equal(t, "2024-02-14T15:00:00", extraction.StartTime)
		assert.Empty(t, extraction.EndTime)
		assert.False(t, extraction.IsAllDay)
		assert.Empty(t, extraction.Timezone)
		assert.Empty(t, extraction.RawText)
	})

	t.Run("handles wrong type for boolean", func(t *testing.T) {
		input := map[string]any{
			"has_datetime": "true", // string instead of bool
			"start_time":   "2024-02-14T15:00:00",
			"confidence":   0.9,
			"reasoning":    "Valid",
		}

		result, err := HandleExtractDateTime(context.Background(), input)
		// Should error because has_datetime will be false (type assertion fails)
		// and start_time is provided (validation triggers)
		assert.NoError(t, err) // No error because has_datetime defaults to false

		var extraction DateTimeExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)
		assert.False(t, extraction.HasDateTime) // defaults to false on bad type
	})

	t.Run("handles wrong type for number", func(t *testing.T) {
		input := map[string]any{
			"has_datetime": false,
			"confidence":   "0.9", // string instead of float64
			"reasoning":    "Valid",
		}

		result, err := HandleExtractDateTime(context.Background(), input)
		require.NoError(t, err)

		var extraction DateTimeExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)

		assert.Equal(t, 0.0, extraction.Confidence) // defaults to 0.0 on bad type
	})
}

func TestHandleExtractDateTime_EmptyInput(t *testing.T) {
	input := map[string]any{}

	result, err := HandleExtractDateTime(context.Background(), input)
	require.NoError(t, err) // Should not error with empty input, just defaults

	var extraction DateTimeExtraction
	err = json.Unmarshal([]byte(result), &extraction)
	require.NoError(t, err)

	assert.False(t, extraction.HasDateTime)
	assert.Empty(t, extraction.StartTime)
	assert.Equal(t, 0.0, extraction.Confidence)
	assert.Empty(t, extraction.Reasoning)
}
