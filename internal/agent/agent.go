package agent

import (
	"context"
	"fmt"
)

// Agent represents an LLM-powered agent with tools
type Agent struct {
	name         string
	apiClient    *APIClient
	registry     *ToolRegistry
	systemPrompt string
}

// AgentConfig configures an agent
type AgentConfig struct {
	Name         string
	APIKey       string
	Model        string
	Temperature  float64
	SystemPrompt string
}

// NewAgent creates a new agent with the given configuration
func NewAgent(cfg AgentConfig) *Agent {
	return &Agent{
		name:         cfg.Name,
		apiClient:    NewAPIClient(cfg.APIKey, cfg.Model, cfg.Temperature),
		registry:     NewToolRegistry(),
		systemPrompt: cfg.SystemPrompt,
	}
}

// Name returns the agent's name
func (a *Agent) Name() string {
	return a.name
}

// RegisterTool adds a tool to the agent
func (a *Agent) RegisterTool(tool Tool, handler ToolHandler) error {
	return a.registry.Register(tool, handler)
}

// MustRegisterTool adds a tool and panics on error
func (a *Agent) MustRegisterTool(tool Tool, handler ToolHandler) {
	a.registry.MustRegister(tool, handler)
}

// Tools returns all registered tools
func (a *Agent) Tools() []Tool {
	return a.registry.Tools()
}

// Execute runs the agent with the given input
func (a *Agent) Execute(ctx context.Context, input AgentInput) (*AgentOutput, error) {
	return a.executeWithPrompt(ctx, input, a.systemPrompt)
}

func (a *Agent) executeWithPrompt(ctx context.Context, input AgentInput, systemPrompt string) (*AgentOutput, error) {
	maxTurns := input.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 1 // Default to single-shot
	}

	messages := make([]Message, len(input.Messages))
	copy(messages, input.Messages)

	var totalUsage UsageStats
	var allToolCalls []ToolCall

	for turn := 0; turn < maxTurns; turn++ {
		// Make API call
		response, err := a.apiClient.Call(ctx, messages, CallOptions{
			System: systemPrompt,
			Tools:  a.registry.Tools(),
		})
		if err != nil {
			return nil, fmt.Errorf("API call failed on turn %d: %w", turn+1, err)
		}
		totalUsage.Add(response.Usage)

		// Check stop reason
		switch response.StopReason {
		case "end_turn":
			// Agent is done - extract final text
			finalText := extractFinalText(response.Content)
			return &AgentOutput{
				ToolCalls:    allToolCalls,
				Conversation: messages,
				Usage:        totalUsage,
				FinalText:    finalText,
			}, nil

		case "tool_use":
			// Process tool calls
			assistantMsg := Message{Role: "assistant", Content: response.Content}
			messages = append(messages, assistantMsg)

			toolResults, toolCalls := a.executeTools(ctx, response.Content)
			allToolCalls = append(allToolCalls, toolCalls...)

			userMsg := Message{Role: "user", Content: toolResults}
			messages = append(messages, userMsg)
			continue

		default:
			return nil, fmt.Errorf("unexpected stop reason: %s", response.StopReason)
		}
	}

	// Max turns exceeded - return what we have
	return &AgentOutput{
		ToolCalls:    allToolCalls,
		Conversation: messages,
		Usage:        totalUsage,
	}, fmt.Errorf("max turns (%d) exceeded", maxTurns)
}

// executeTools runs all tool_use blocks and returns results
func (a *Agent) executeTools(ctx context.Context, content []ContentBlock) ([]ContentBlock, []ToolCall) {
	var results []ContentBlock
	var calls []ToolCall

	for _, block := range content {
		toolUse, ok := block.(ToolUseBlock)
		if !ok {
			continue
		}

		output, err := a.registry.Execute(ctx, toolUse.Name, toolUse.Input)

		call := ToolCall{
			Name:   toolUse.Name,
			Input:  toolUse.Input,
			Output: output,
			Error:  err,
		}
		calls = append(calls, call)

		resultBlock := ToolResultBlock{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   output,
			IsError:   err != nil,
		}
		if err != nil {
			resultBlock.Content = err.Error()
		}
		results = append(results, resultBlock)
	}

	return results, calls
}

// extractFinalText extracts text from the final response
func extractFinalText(content []ContentBlock) string {
	for _, block := range content {
		if text, ok := block.(TextBlock); ok {
			return text.Text
		}
	}
	return ""
}

// ExecuteSingleTool runs the agent expecting exactly one tool call
func (a *Agent) ExecuteSingleTool(ctx context.Context, userMessage string) (*ToolCall, error) {
	input := AgentInput{
		Messages: []Message{
			{
				Role:    "user",
				Content: []ContentBlock{TextBlock{Type: "text", Text: userMessage}},
			},
		},
		MaxTurns: 1,
	}

	output, err := a.Execute(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(output.ToolCalls) == 0 {
		return nil, fmt.Errorf("no tool was called")
	}

	return &output.ToolCalls[0], nil
}

// IsConfigured returns true if the agent's API client is configured
func (a *Agent) IsConfigured() bool {
	return a.apiClient != nil && a.apiClient.IsConfigured()
}
