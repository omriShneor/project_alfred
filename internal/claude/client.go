package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
)

const (
	defaultAPIURL     = "https://api.anthropic.com/v1/messages"
	defaultModel      = "claude-sonnet-4-20250514"
	defaultMaxTokens  = 1024
	anthropicVersion  = "2023-06-01"
)

// Client is a Claude API client for event detection
type Client struct {
	apiKey      string
	model       string
	apiURL      string
	httpClient  *http.Client
	temperature float64
}

// NewClient creates a new Claude API client
func NewClient(apiKey, model string, temperature float64) *Client {
	if model == "" {
		model = defaultModel
	}
	if temperature <= 0 {
		temperature = 0.1
	}

	return &Client{
		apiKey:      apiKey,
		model:       model,
		apiURL:      defaultAPIURL,
		temperature: temperature,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// EventAnalysis represents Claude's analysis of messages for calendar events
type EventAnalysis struct {
	HasEvent   bool       `json:"has_event"`
	Action     string     `json:"action"` // "create", "update", "delete", "none"
	Event      *EventData `json:"event,omitempty"`
	Reasoning  string     `json:"reasoning"`
	Confidence float64    `json:"confidence"`
}

// EventData contains the extracted event details
type EventData struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	StartTime   string `json:"start_time"` // ISO 8601 format
	EndTime     string `json:"end_time,omitempty"`
	Location    string `json:"location,omitempty"`
	UpdateRef   string `json:"update_ref,omitempty"` // Google event ID for updates/deletes
}

// anthropicRequest represents the API request structure
type anthropicRequest struct {
	Model       string              `json:"model"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature"`
	System      string              `json:"system"`
	Messages    []anthropicMessage  `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse represents the API response structure
type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// MessageContext represents a message for analysis
type MessageContext struct {
	Sender    string    `json:"sender"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// ExistingEvent represents an existing synced event for context
type ExistingEvent struct {
	GoogleEventID string     `json:"google_event_id"`
	Title         string     `json:"title"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Location      string     `json:"location,omitempty"`
}

// AnalyzeMessages sends message history to Claude for event detection
func (c *Client) AnalyzeMessages(
	ctx context.Context,
	history []database.MessageRecord,
	newMessage database.MessageRecord,
	existingEvents []database.CalendarEvent,
) (*EventAnalysis, error) {
	// Build the user prompt with context
	userPrompt := c.buildUserPrompt(history, newMessage, existingEvents)

	// Create the API request
	req := anthropicRequest{
		Model:       c.model,
		MaxTokens:   defaultMaxTokens,
		Temperature: c.temperature,
		System:      SystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	// Parse the JSON response from Claude
	var analysis EventAnalysis
	responseText := apiResp.Content[0].Text

	// Try to extract JSON from the response (Claude might wrap it in markdown)
	jsonStr := extractJSON(responseText)
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analysis JSON: %w (response: %s)", err, responseText)
	}

	return &analysis, nil
}

// buildUserPrompt constructs the prompt with message history and context
func (c *Client) buildUserPrompt(
	history []database.MessageRecord,
	newMessage database.MessageRecord,
	existingEvents []database.CalendarEvent,
) string {
	var prompt bytes.Buffer

	prompt.WriteString("## Message History (last messages from this channel)\n\n")

	for _, msg := range history {
		prompt.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			msg.Timestamp.Format("2006-01-02 15:04"),
			msg.SenderName,
			msg.MessageText,
		))
	}

	prompt.WriteString("\n## New Message (just received)\n\n")
	prompt.WriteString(fmt.Sprintf("[%s] %s: %s\n",
		newMessage.Timestamp.Format("2006-01-02 15:04"),
		newMessage.SenderName,
		newMessage.MessageText,
	))

	if len(existingEvents) > 0 {
		prompt.WriteString("\n## Existing Calendar Events for this channel\n\n")
		for _, event := range existingEvents {
			endStr := ""
			if event.EndTime != nil {
				endStr = fmt.Sprintf(" - %s", event.EndTime.Format("2006-01-02 15:04"))
			}
			googleID := ""
			if event.GoogleEventID != nil {
				googleID = *event.GoogleEventID
			}
			prompt.WriteString(fmt.Sprintf("- [ID: %s] %s @ %s%s",
				googleID,
				event.Title,
				event.StartTime.Format("2006-01-02 15:04"),
				endStr,
			))
			if event.Location != "" {
				prompt.WriteString(fmt.Sprintf(" (Location: %s)", event.Location))
			}
			prompt.WriteString("\n")
		}
	} else {
		prompt.WriteString("\n## Existing Calendar Events for this channel\n\nNo existing events.\n")
	}

	prompt.WriteString("\n## Current Date/Time Reference\n\n")
	prompt.WriteString(fmt.Sprintf("Current time: %s\n", time.Now().Format("2006-01-02 15:04 (Monday)")))

	prompt.WriteString("\nAnalyze these messages and respond with your JSON analysis.")

	return prompt.String()
}

// extractJSON attempts to extract JSON from a response that might be wrapped in markdown
func extractJSON(text string) string {
	// Try to find JSON block in markdown code fence
	start := 0
	if idx := findJSONStart(text); idx >= 0 {
		start = idx
	}

	end := len(text)
	if idx := findJSONEnd(text, start); idx >= 0 {
		end = idx + 1
	}

	return text[start:end]
}

func findJSONStart(text string) int {
	// Look for opening brace, possibly after ```json
	for i := 0; i < len(text); i++ {
		if text[i] == '{' {
			return i
		}
	}
	return -1
}

func findJSONEnd(text string, start int) int {
	// Find matching closing brace
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// IsConfigured returns true if the client has an API key
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}
