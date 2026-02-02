package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultAPIURL        = "https://api.anthropic.com/v1/messages"
	defaultModel         = "claude-sonnet-4-20250514"
	defaultMaxTokens     = 4096
	anthropicVersion     = "2023-06-01"
	anthropicBetaHeader  = "tools-2024-04-04"
)

// APIClient handles communication with the Anthropic API
type APIClient struct {
	apiKey      string
	model       string
	apiURL      string
	httpClient  *http.Client
	temperature float64
}

// NewAPIClient creates a new Anthropic API client
func NewAPIClient(apiKey, model string, temperature float64) *APIClient {
	if model == "" {
		model = defaultModel
	}
	if temperature <= 0 {
		temperature = 0.1
	}

	return &APIClient{
		apiKey:      apiKey,
		model:       model,
		apiURL:      defaultAPIURL,
		temperature: temperature,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for tool use
		},
	}
}

// apiRequest represents the Anthropic API request with tools
type apiRequest struct {
	Model       string                   `json:"model"`
	MaxTokens   int                      `json:"max_tokens"`
	Temperature float64                  `json:"temperature"`
	System      string                   `json:"system,omitempty"`
	Tools       []map[string]any `json:"tools,omitempty"`
	ToolChoice  *toolChoice              `json:"tool_choice,omitempty"`
	Messages    []apiMessage             `json:"messages"`
}

type toolChoice struct {
	Type string `json:"type"` // "auto", "any", or "tool"
	Name string `json:"name,omitempty"` // Only for type="tool"
}

type apiMessage struct {
	Role    string      `json:"role"`
	Content any `json:"content"` // string or []ContentBlock
}

// apiResponse represents the Anthropic API response
type apiResponse struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Role       string            `json:"role"`
	Content    []apiContentBlock `json:"content"`
	StopReason string            `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type apiContentBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

// APIResponse wraps the parsed response from the API
type APIResponse struct {
	Content    []ContentBlock
	StopReason string
	Usage      UsageStats
}

// CallOptions configures an API call
type CallOptions struct {
	System     string
	Tools      []Tool
	ToolChoice string // "auto", "any", or specific tool name
	MaxTokens  int
}

// Call makes a request to the Anthropic API
func (c *APIClient) Call(ctx context.Context, messages []Message, opts CallOptions) (*APIResponse, error) {
	// Convert messages to API format
	apiMessages := make([]apiMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = apiMessage{
			Role:    msg.Role,
			Content: convertContentToAPI(msg.Content),
		}
	}

	// Convert tools to API format
	var apiTools []map[string]any
	if len(opts.Tools) > 0 {
		apiTools = make([]map[string]any, len(opts.Tools))
		for i, tool := range opts.Tools {
			apiTools[i] = map[string]any{
				"name":         tool.Name,
				"description":  tool.Description,
				"input_schema": tool.InputSchema,
			}
		}
	}

	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	req := apiRequest{
		Model:       c.model,
		MaxTokens:   maxTokens,
		Temperature: c.temperature,
		System:      opts.System,
		Tools:       apiTools,
		Messages:    apiMessages,
	}

	// Set tool choice if specified
	if opts.ToolChoice != "" && len(opts.Tools) > 0 {
		switch opts.ToolChoice {
		case "auto", "any":
			req.ToolChoice = &toolChoice{Type: opts.ToolChoice}
		default:
			// Specific tool name
			req.ToolChoice = &toolChoice{Type: "tool", Name: opts.ToolChoice}
		}
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	if len(opts.Tools) > 0 {
		httpReq.Header.Set("anthropic-beta", anthropicBetaHeader)
	}

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

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	// Convert response to our types
	content := make([]ContentBlock, len(apiResp.Content))
	for i, block := range apiResp.Content {
		switch block.Type {
		case "text":
			content[i] = TextBlock{Type: "text", Text: block.Text}
		case "tool_use":
			content[i] = ToolUseBlock{
				Type:  "tool_use",
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			}
		}
	}

	return &APIResponse{
		Content:    content,
		StopReason: apiResp.StopReason,
		Usage: UsageStats{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
			TotalTokens:  apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
	}, nil
}

// convertContentToAPI converts ContentBlock slice to API format
func convertContentToAPI(content []ContentBlock) any {
	if len(content) == 1 {
		if text, ok := content[0].(TextBlock); ok {
			return text.Text
		}
	}

	result := make([]map[string]any, len(content))
	for i, block := range content {
		switch b := block.(type) {
		case TextBlock:
			result[i] = map[string]any{
				"type": "text",
				"text": b.Text,
			}
		case ToolUseBlock:
			result[i] = map[string]any{
				"type":  "tool_use",
				"id":    b.ID,
				"name":  b.Name,
				"input": b.Input,
			}
		case ToolResultBlock:
			block := map[string]any{
				"type":        "tool_result",
				"tool_use_id": b.ToolUseID,
				"content":     b.Content,
			}
			if b.IsError {
				block["is_error"] = true
			}
			result[i] = block
		}
	}
	return result
}

// IsConfigured returns true if the client has an API key
func (c *APIClient) IsConfigured() bool {
	return c.apiKey != ""
}
