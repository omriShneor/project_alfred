package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name            string
		apiKey          string
		model           string
		temperature     float64
		expectedModel   string
		expectedTemp    float64
		expectedConfig  bool
	}{
		{
			name:           "with all parameters",
			apiKey:         "test-api-key",
			model:          "claude-3-opus",
			temperature:    0.5,
			expectedModel:  "claude-3-opus",
			expectedTemp:   0.5,
			expectedConfig: true,
		},
		{
			name:           "empty model uses default",
			apiKey:         "test-api-key",
			model:          "",
			temperature:    0.3,
			expectedModel:  defaultModel,
			expectedTemp:   0.3,
			expectedConfig: true,
		},
		{
			name:           "zero temperature uses default",
			apiKey:         "test-api-key",
			model:          "claude-3-sonnet",
			temperature:    0,
			expectedModel:  "claude-3-sonnet",
			expectedTemp:   0.1,
			expectedConfig: true,
		},
		{
			name:           "negative temperature uses default",
			apiKey:         "test-api-key",
			model:          "custom-model",
			temperature:    -0.5,
			expectedModel:  "custom-model",
			expectedTemp:   0.1,
			expectedConfig: true,
		},
		{
			name:           "empty api key",
			apiKey:         "",
			model:          "some-model",
			temperature:    0.2,
			expectedModel:  "some-model",
			expectedTemp:   0.2,
			expectedConfig: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey, tt.model, tt.temperature)

			require.NotNil(t, client)
			assert.Equal(t, tt.expectedModel, client.model)
			assert.Equal(t, tt.expectedTemp, client.temperature)
			assert.Equal(t, tt.expectedConfig, client.IsConfigured())
		})
	}
}

func TestIsConfigured(t *testing.T) {
	t.Run("configured with api key", func(t *testing.T) {
		client := NewClient("test-key", "", 0)
		assert.True(t, client.IsConfigured())
	})

	t.Run("not configured without api key", func(t *testing.T) {
		client := NewClient("", "", 0)
		assert.False(t, client.IsConfigured())
	})
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean json",
			input:    `{"has_event": true, "action": "create"}`,
			expected: `{"has_event": true, "action": "create"}`,
		},
		{
			name:     "json in markdown fence",
			input:    "```json\n{\"has_event\": true}\n```",
			expected: `{"has_event": true}`,
		},
		{
			name:     "json with text before",
			input:    "Here is the analysis:\n{\"has_event\": false}",
			expected: `{"has_event": false}`,
		},
		{
			name:     "json with text after",
			input:    "{\"has_event\": true}\nThis is my analysis.",
			expected: `{"has_event": true}`,
		},
		{
			name:     "json with text before and after",
			input:    "Analysis:\n{\"action\": \"none\"}\nDone.",
			expected: `{"action": "none"}`,
		},
		{
			name:     "nested json objects",
			input:    `{"has_event": true, "event": {"title": "Meeting", "nested": {"deep": true}}}`,
			expected: `{"has_event": true, "event": {"title": "Meeting", "nested": {"deep": true}}}`,
		},
		{
			name:     "json with arrays",
			input:    `{"items": [1, 2, {"key": "value"}]}`,
			expected: `{"items": [1, 2, {"key": "value"}]}`,
		},
		{
			name: "complex markdown response",
			input: `Sure, I'll analyze this message.

` + "```json" + `
{
  "has_event": true,
  "action": "create",
  "event": {
    "title": "Team Meeting",
    "start_time": "2024-01-15T14:00:00Z"
  }
}
` + "```" + `

The message clearly indicates a meeting.`,
			expected: `{
  "has_event": true,
  "action": "create",
  "event": {
    "title": "Team Meeting",
    "start_time": "2024-01-15T14:00:00Z"
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindJSONStart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "starts with brace",
			input:    `{"key": "value"}`,
			expected: 0,
		},
		{
			name:     "brace after text",
			input:    "prefix {\"key\": \"value\"}",
			expected: 7,
		},
		{
			name:     "brace after markdown",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: 8,
		},
		{
			name:     "no brace",
			input:    "no json here",
			expected: -1,
		},
		{
			name:     "empty string",
			input:    "",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJSONStart(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindJSONEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		expected int
	}{
		{
			name:     "simple object",
			input:    `{"key": "value"}`,
			start:    0,
			expected: 15,
		},
		{
			name:     "nested objects",
			input:    `{"outer": {"inner": {}}}`,
			start:    0,
			expected: 23,
		},
		{
			name:     "with trailing text",
			input:    `{"key": "value"} extra`,
			start:    0,
			expected: 15,
		},
		{
			name:     "unmatched braces",
			input:    `{"key": "value"`,
			start:    0,
			expected: -1,
		},
		{
			name:     "start in middle",
			input:    `prefix {"key": "value"}`,
			start:    7,
			expected: 22,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJSONEnd(tt.input, tt.start)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateEmailBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		maxLen   int
		expected string
	}{
		{
			name:     "under limit unchanged",
			body:     "short body",
			maxLen:   100,
			expected: "short body",
		},
		{
			name:     "at limit unchanged",
			body:     "exact",
			maxLen:   5,
			expected: "exact",
		},
		{
			name:     "over limit truncated",
			body:     "this is a longer body that exceeds the limit",
			maxLen:   10,
			expected: "this is a ...",
		},
		{
			name:     "empty body",
			body:     "",
			maxLen:   100,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateEmailBody(tt.body, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildUserPrompt(t *testing.T) {
	client := NewClient("test-key", "", 0)

	t.Run("with message history", func(t *testing.T) {
		history := []database.MessageRecord{
			{
				SenderName:  "Alice",
				MessageText: "Let's meet tomorrow at 2pm",
				Timestamp:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			{
				SenderName:  "Bob",
				MessageText: "Sounds good!",
				Timestamp:   time.Date(2024, 1, 15, 10, 5, 0, 0, time.UTC),
			},
		}

		newMessage := database.MessageRecord{
			SenderName:  "Alice",
			MessageText: "See you in the conference room",
			Timestamp:   time.Date(2024, 1, 15, 10, 10, 0, 0, time.UTC),
		}

		prompt := client.buildUserPrompt(history, newMessage, nil)

		assert.Contains(t, prompt, "Message History")
		assert.Contains(t, prompt, "Alice")
		assert.Contains(t, prompt, "Let's meet tomorrow at 2pm")
		assert.Contains(t, prompt, "Bob")
		assert.Contains(t, prompt, "Sounds good!")
		assert.Contains(t, prompt, "New Message")
		assert.Contains(t, prompt, "See you in the conference room")
		assert.Contains(t, prompt, "No existing events")
	})

	t.Run("with existing events", func(t *testing.T) {
		history := []database.MessageRecord{}
		newMessage := database.MessageRecord{
			SenderName:  "User",
			MessageText: "Update the meeting",
			Timestamp:   time.Now(),
		}

		endTime := time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC)
		googleID := "google-123"
		existingEvents := []database.CalendarEvent{
			{
				ID:            1,
				GoogleEventID: &googleID,
				Title:         "Existing Meeting",
				StartTime:     time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
				EndTime:       &endTime,
				Location:      "Room A",
				Status:        database.EventStatusSynced,
			},
		}

		prompt := client.buildUserPrompt(history, newMessage, existingEvents)

		assert.Contains(t, prompt, "Existing Calendar Events")
		assert.Contains(t, prompt, "Existing Meeting")
		assert.Contains(t, prompt, "google-123")
		assert.Contains(t, prompt, "Room A")
		assert.Contains(t, prompt, "synced")
	})

	t.Run("empty history", func(t *testing.T) {
		newMessage := database.MessageRecord{
			SenderName:  "Solo User",
			MessageText: "First message",
			Timestamp:   time.Now(),
		}

		prompt := client.buildUserPrompt(nil, newMessage, nil)

		assert.Contains(t, prompt, "Message History")
		assert.Contains(t, prompt, "New Message")
		assert.Contains(t, prompt, "Solo User")
		assert.Contains(t, prompt, "First message")
	})
}

func TestBuildEmailPrompt(t *testing.T) {
	client := NewClient("test-key", "", 0)

	t.Run("email without thread history", func(t *testing.T) {
		email := EmailContent{
			Subject: "Meeting Request",
			From:    "sender@example.com",
			To:      "recipient@example.com",
			Date:    "2024-01-15T10:00:00Z",
			Body:    "Please join us for a meeting on Friday at 3pm.",
		}

		prompt := client.buildEmailPrompt(email)

		assert.Contains(t, prompt, "Email to Analyze")
		assert.Contains(t, prompt, "Meeting Request")
		assert.Contains(t, prompt, "sender@example.com")
		assert.Contains(t, prompt, "recipient@example.com")
		assert.Contains(t, prompt, "Please join us for a meeting")
		assert.NotContains(t, prompt, "Email Thread History")
	})

	t.Run("email with thread history", func(t *testing.T) {
		email := EmailContent{
			Subject: "Re: Meeting Request",
			From:    "reply@example.com",
			To:      "original@example.com",
			Date:    "2024-01-15T11:00:00Z",
			Body:    "I'll be there!",
			ThreadHistory: []EmailThreadMessage{
				{
					From:    "original@example.com",
					Date:    "2024-01-15T10:00:00Z",
					Subject: "Meeting Request",
					Body:    "Let's meet on Friday",
				},
			},
		}

		prompt := client.buildEmailPrompt(email)

		assert.Contains(t, prompt, "Email Thread History")
		assert.Contains(t, prompt, "original@example.com")
		assert.Contains(t, prompt, "Let's meet on Friday")
		assert.Contains(t, prompt, "Email to Analyze")
		assert.Contains(t, prompt, "I'll be there!")
	})

	t.Run("truncates long email body", func(t *testing.T) {
		longBody := ""
		for i := 0; i < 10000; i++ {
			longBody += "x"
		}

		email := EmailContent{
			Subject: "Long Email",
			From:    "sender@example.com",
			To:      "recipient@example.com",
			Date:    "2024-01-15T10:00:00Z",
			Body:    longBody,
		}

		prompt := client.buildEmailPrompt(email)

		assert.Contains(t, prompt, "[... content truncated ...]")
		assert.Less(t, len(prompt), len(longBody)) // Prompt should be shorter than original body
	})
}

func TestAnalyzeMessages_Success(t *testing.T) {
	// Create a mock server
	mockResponse := anthropicResponse{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			{
				Type: "text",
				Text: `{"has_event": true, "action": "create", "event": {"title": "Team Meeting", "start_time": "2024-01-15T14:00:00Z"}, "reasoning": "User mentioned a meeting", "confidence": 0.95}`,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, anthropicVersion, r.Header.Get("anthropic-version"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := &Client{
		apiKey:      "test-api-key",
		model:       "test-model",
		apiURL:      server.URL,
		temperature: 0.1,
		httpClient:  &http.Client{},
	}

	history := []database.MessageRecord{
		{SenderName: "Alice", MessageText: "Let's have a meeting tomorrow", Timestamp: time.Now()},
	}
	newMessage := database.MessageRecord{
		SenderName:  "Alice",
		MessageText: "At 2pm in the conference room",
		Timestamp:   time.Now(),
	}

	analysis, err := client.AnalyzeMessages(context.Background(), history, newMessage, nil)

	require.NoError(t, err)
	require.NotNil(t, analysis)
	assert.True(t, analysis.HasEvent)
	assert.Equal(t, "create", analysis.Action)
	assert.NotNil(t, analysis.Event)
	assert.Equal(t, "Team Meeting", analysis.Event.Title)
	assert.Equal(t, "2024-01-15T14:00:00Z", analysis.Event.StartTime)
	assert.Equal(t, 0.95, analysis.Confidence)
}

func TestAnalyzeMessages_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"type": "server_error", "message": "Internal error"}}`))
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		model:      "test-model",
		apiURL:     server.URL,
		httpClient: &http.Client{},
	}

	_, err := client.AnalyzeMessages(context.Background(), nil, database.MessageRecord{}, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
	assert.Contains(t, err.Error(), "500")
}

func TestAnalyzeMessages_EmptyResponse(t *testing.T) {
	mockResponse := anthropicResponse{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		model:      "test-model",
		apiURL:     server.URL,
		httpClient: &http.Client{},
	}

	_, err := client.AnalyzeMessages(context.Background(), nil, database.MessageRecord{}, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

func TestAnalyzeMessages_InvalidJSON(t *testing.T) {
	mockResponse := anthropicResponse{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			{Type: "text", Text: "This is not valid JSON"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		model:      "test-model",
		apiURL:     server.URL,
		httpClient: &http.Client{},
	}

	_, err := client.AnalyzeMessages(context.Background(), nil, database.MessageRecord{}, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse analysis JSON")
}

func TestAnalyzeMessages_MarkdownWrappedJSON(t *testing.T) {
	mockResponse := anthropicResponse{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			{
				Type: "text",
				Text: "Here's my analysis:\n```json\n{\"has_event\": true, \"action\": \"create\", \"event\": {\"title\": \"Meeting\"}, \"reasoning\": \"test\", \"confidence\": 0.9}\n```",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		model:      "test-model",
		apiURL:     server.URL,
		httpClient: &http.Client{},
	}

	analysis, err := client.AnalyzeMessages(context.Background(), nil, database.MessageRecord{}, nil)

	require.NoError(t, err)
	require.NotNil(t, analysis)
	assert.True(t, analysis.HasEvent)
	assert.Equal(t, "Meeting", analysis.Event.Title)
}

func TestAnalyzeEmail_Success(t *testing.T) {
	mockResponse := anthropicResponse{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			{
				Type: "text",
				Text: `{"has_event": true, "action": "create", "event": {"title": "Conference Call", "start_time": "2024-01-20T10:00:00Z"}, "reasoning": "Email invitation", "confidence": 0.85}`,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		model:      "test-model",
		apiURL:     server.URL,
		httpClient: &http.Client{},
	}

	email := EmailContent{
		Subject: "Conference Call Invite",
		From:    "organizer@example.com",
		To:      "attendee@example.com",
		Date:    "2024-01-15T10:00:00Z",
		Body:    "You're invited to a conference call on January 20th at 10am.",
	}

	analysis, err := client.AnalyzeEmail(context.Background(), email)

	require.NoError(t, err)
	require.NotNil(t, analysis)
	assert.True(t, analysis.HasEvent)
	assert.Equal(t, "create", analysis.Action)
	assert.Equal(t, "Conference Call", analysis.Event.Title)
	assert.Equal(t, 0.85, analysis.Confidence)
}

func TestAnalyzeEmail_NoEvent(t *testing.T) {
	mockResponse := anthropicResponse{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			{
				Type: "text",
				Text: `{"has_event": false, "action": "none", "reasoning": "Just a newsletter", "confidence": 0.99}`,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		model:      "test-model",
		apiURL:     server.URL,
		httpClient: &http.Client{},
	}

	email := EmailContent{
		Subject: "Weekly Newsletter",
		From:    "newsletter@company.com",
		To:      "subscriber@example.com",
		Date:    "2024-01-15T10:00:00Z",
		Body:    "Here's what happened this week...",
	}

	analysis, err := client.AnalyzeEmail(context.Background(), email)

	require.NoError(t, err)
	require.NotNil(t, analysis)
	assert.False(t, analysis.HasEvent)
	assert.Equal(t, "none", analysis.Action)
	assert.Nil(t, analysis.Event)
}
