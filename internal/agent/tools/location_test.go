package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleExtractLocation_ValidExtractionWithLocation(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected LocationExtraction
	}{
		{
			name: "physical location with address",
			input: map[string]any{
				"has_location": true,
				"name":         "Starbucks",
				"address":      "123 Main St, New York, NY 10001",
				"type":         "physical",
				"confidence":   0.95,
				"raw_text":     "Meet at Starbucks on 123 Main St",
				"reasoning":    "Specific venue name and full address provided",
			},
			expected: LocationExtraction{
				HasLocation: true,
				Name:        "Starbucks",
				Address:     "123 Main St, New York, NY 10001",
				Type:        "physical",
				Confidence:  0.95,
				RawText:     "Meet at Starbucks on 123 Main St",
				Reasoning:   "Specific venue name and full address provided",
			},
		},
		{
			name: "virtual location with URL",
			input: map[string]any{
				"has_location": true,
				"name":         "Zoom Meeting",
				"type":         "virtual",
				"url":          "https://zoom.us/j/123456789",
				"confidence":   1.0,
				"raw_text":     "Join via Zoom: https://zoom.us/j/123456789",
				"reasoning":    "Virtual meeting with Zoom URL",
			},
			expected: LocationExtraction{
				HasLocation: true,
				Name:        "Zoom Meeting",
				Type:        "virtual",
				URL:         "https://zoom.us/j/123456789",
				Confidence:  1.0,
				RawText:     "Join via Zoom: https://zoom.us/j/123456789",
				Reasoning:   "Virtual meeting with Zoom URL",
			},
		},
		{
			name: "location name only",
			input: map[string]any{
				"has_location": true,
				"name":         "Conference Room A",
				"type":         "physical",
				"confidence":   0.85,
				"reasoning":    "Office location mentioned",
			},
			expected: LocationExtraction{
				HasLocation: true,
				Name:        "Conference Room A",
				Type:        "physical",
				Confidence:  0.85,
				Reasoning:   "Office location mentioned",
			},
		},
		{
			name: "contextual location reference",
			input: map[string]any{
				"has_location": true,
				"name":         "Sarah's place",
				"type":         "unknown",
				"confidence":   0.7,
				"raw_text":     "Let's meet at Sarah's place",
				"reasoning":    "Contextual reference without specific address",
			},
			expected: LocationExtraction{
				HasLocation: true,
				Name:        "Sarah's place",
				Type:        "unknown",
				Confidence:  0.7,
				RawText:     "Let's meet at Sarah's place",
				Reasoning:   "Contextual reference without specific address",
			},
		},
		{
			name: "Google Meet URL",
			input: map[string]any{
				"has_location": true,
				"name":         "Google Meet",
				"type":         "virtual",
				"url":          "https://meet.google.com/abc-defg-hij",
				"confidence":   1.0,
				"reasoning":    "Google Meet link detected",
			},
			expected: LocationExtraction{
				HasLocation: true,
				Name:        "Google Meet",
				Type:        "virtual",
				URL:         "https://meet.google.com/abc-defg-hij",
				Confidence:  1.0,
				Reasoning:   "Google Meet link detected",
			},
		},
		{
			name: "minimal valid location",
			input: map[string]any{
				"has_location": true,
				"name":         "The Office",
				"confidence":   0.8,
				"reasoning":    "Location mentioned",
			},
			expected: LocationExtraction{
				HasLocation: true,
				Name:        "The Office",
				Confidence:  0.8,
				Reasoning:   "Location mentioned",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleExtractLocation(context.Background(), tt.input)
			require.NoError(t, err)

			var extraction LocationExtraction
			err = json.Unmarshal([]byte(result), &extraction)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.HasLocation, extraction.HasLocation)
			assert.Equal(t, tt.expected.Name, extraction.Name)
			assert.Equal(t, tt.expected.Address, extraction.Address)
			assert.Equal(t, tt.expected.Type, extraction.Type)
			assert.Equal(t, tt.expected.URL, extraction.URL)
			assert.Equal(t, tt.expected.Confidence, extraction.Confidence)
			assert.Equal(t, tt.expected.RawText, extraction.RawText)
			assert.Equal(t, tt.expected.Reasoning, extraction.Reasoning)
		})
	}
}

func TestHandleExtractLocation_ValidExtractionWithNoLocation(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected LocationExtraction
	}{
		{
			name: "no location mentioned",
			input: map[string]any{
				"has_location": false,
				"confidence":   0.95,
				"reasoning":    "No location information in text",
				"raw_text":     "Let's have a call tomorrow",
			},
			expected: LocationExtraction{
				HasLocation: false,
				Confidence:  0.95,
				Reasoning:   "No location information in text",
				RawText:     "Let's have a call tomorrow",
			},
		},
		{
			name: "vague location reference",
			input: map[string]any{
				"has_location": false,
				"confidence":   0.6,
				"reasoning":    "Reference to 'somewhere' is too vague",
			},
			expected: LocationExtraction{
				HasLocation: false,
				Confidence:  0.6,
				Reasoning:   "Reference to 'somewhere' is too vague",
			},
		},
		{
			name: "minimal no location",
			input: map[string]any{
				"has_location": false,
				"confidence":   1.0,
				"reasoning":    "No location data",
			},
			expected: LocationExtraction{
				HasLocation: false,
				Confidence:  1.0,
				Reasoning:   "No location data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleExtractLocation(context.Background(), tt.input)
			require.NoError(t, err)

			var extraction LocationExtraction
			err = json.Unmarshal([]byte(result), &extraction)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.HasLocation, extraction.HasLocation)
			assert.Empty(t, extraction.Name)
			assert.Empty(t, extraction.Address)
			assert.Empty(t, extraction.Type)
			assert.Empty(t, extraction.URL)
			assert.Equal(t, tt.expected.Confidence, extraction.Confidence)
			assert.Equal(t, tt.expected.Reasoning, extraction.Reasoning)
		})
	}
}

func TestHandleExtractLocation_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]any
		expectedErr string
	}{
		{
			name: "has_location true but no name",
			input: map[string]any{
				"has_location": true,
				"address":      "123 Main St",
				"confidence":   0.8,
				"reasoning":    "Address provided but no name",
			},
			expectedErr: "name is required when has_location is true",
		},
		{
			name: "has_location true with empty name",
			input: map[string]any{
				"has_location": true,
				"name":         "",
				"address":      "123 Main St",
				"confidence":   0.8,
				"reasoning":    "Empty name",
			},
			expectedErr: "name is required when has_location is true",
		},
		{
			name: "has_location true with missing name field",
			input: map[string]any{
				"has_location": true,
				"type":         "physical",
				"confidence":   0.8,
				"reasoning":    "No name field",
			},
			expectedErr: "name is required when has_location is true",
		},
		{
			name: "has_location true with only URL",
			input: map[string]any{
				"has_location": true,
				"url":          "https://zoom.us/j/123456789",
				"confidence":   0.9,
				"reasoning":    "URL without name",
			},
			expectedErr: "name is required when has_location is true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleExtractLocation(context.Background(), tt.input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Empty(t, result)
		})
	}
}

func TestHandleExtractLocation_TypeConversion(t *testing.T) {
	t.Run("handles missing optional fields gracefully", func(t *testing.T) {
		input := map[string]any{
			"has_location": true,
			"name":         "Main Office",
			"confidence":   0.9,
			"reasoning":    "Valid",
			// address, type, url, raw_text omitted
		}

		result, err := HandleExtractLocation(context.Background(), input)
		require.NoError(t, err)

		var extraction LocationExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)

		assert.True(t, extraction.HasLocation)
		assert.Equal(t, "Main Office", extraction.Name)
		assert.Empty(t, extraction.Address)
		assert.Empty(t, extraction.Type)
		assert.Empty(t, extraction.URL)
		assert.Empty(t, extraction.RawText)
	})

	t.Run("handles wrong type for boolean", func(t *testing.T) {
		input := map[string]any{
			"has_location": "true", // string instead of bool
			"name":         "Office",
			"confidence":   0.9,
			"reasoning":    "Valid",
		}

		result, err := HandleExtractLocation(context.Background(), input)
		require.NoError(t, err) // No error because has_location defaults to false

		var extraction LocationExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)
		assert.False(t, extraction.HasLocation) // defaults to false on bad type
	})

	t.Run("handles wrong type for number", func(t *testing.T) {
		input := map[string]any{
			"has_location": false,
			"confidence":   "0.9", // string instead of float64
			"reasoning":    "Valid",
		}

		result, err := HandleExtractLocation(context.Background(), input)
		require.NoError(t, err)

		var extraction LocationExtraction
		err = json.Unmarshal([]byte(result), &extraction)
		require.NoError(t, err)

		assert.Equal(t, 0.0, extraction.Confidence) // defaults to 0.0 on bad type
	})
}

func TestHandleExtractLocation_EmptyInput(t *testing.T) {
	input := map[string]any{}

	result, err := HandleExtractLocation(context.Background(), input)
	require.NoError(t, err) // Should not error with empty input, just defaults

	var extraction LocationExtraction
	err = json.Unmarshal([]byte(result), &extraction)
	require.NoError(t, err)

	assert.False(t, extraction.HasLocation)
	assert.Empty(t, extraction.Name)
	assert.Equal(t, 0.0, extraction.Confidence)
	assert.Empty(t, extraction.Reasoning)
}
