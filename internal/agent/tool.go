package agent

import (
	"context"
	"fmt"
	"sync"
)

// Tool represents a function that an agent can call
type Tool struct {
	// Name is the unique identifier for this tool
	Name string `json:"name"`

	// Description explains what the tool does (3-4+ sentences recommended)
	Description string `json:"description"`

	// InputSchema defines the expected input format (JSON Schema)
	InputSchema map[string]any `json:"input_schema"`
}

// ToolHandler is a function that executes a tool and returns the result
type ToolHandler func(ctx context.Context, input map[string]any) (string, error)

// ToolRegistry manages tools and their handlers
type ToolRegistry struct {
	tools    []Tool
	handlers map[string]ToolHandler
	mu       sync.RWMutex
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:    make([]Tool, 0),
		handlers: make(map[string]ToolHandler),
	}
}

// Register adds a tool with its handler to the registry
func (r *ToolRegistry) Register(tool Tool, handler ToolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[tool.Name]; exists {
		return fmt.Errorf("tool already registered: %s", tool.Name)
	}

	r.tools = append(r.tools, tool)
	r.handlers[tool.Name] = handler
	return nil
}

// MustRegister registers a tool and panics on error
func (r *ToolRegistry) MustRegister(tool Tool, handler ToolHandler) {
	if err := r.Register(tool, handler); err != nil {
		panic(err)
	}
}

// Execute runs a tool by name with the given input
func (r *ToolRegistry) Execute(ctx context.Context, name string, input map[string]any) (string, error) {
	r.mu.RLock()
	handler, ok := r.handlers[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	return handler(ctx, input)
}

// Tools returns all registered tools
func (r *ToolRegistry) Tools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Tool, len(r.tools))
	copy(result, r.tools)
	return result
}

// HasTool checks if a tool is registered
func (r *ToolRegistry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.handlers[name]
	return ok
}

// ToolCount returns the number of registered tools
func (r *ToolRegistry) ToolCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// BuildJSONSchema is a helper to construct JSON Schema objects
func BuildJSONSchema(schemaType string, properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		"type":       schemaType,
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// PropertyString creates a string property definition
func PropertyString(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

// PropertyInt creates an integer property definition
func PropertyInt(description string) map[string]any {
	return map[string]any{
		"type":        "integer",
		"description": description,
	}
}

// PropertyNumber creates a number property definition
func PropertyNumber(description string) map[string]any {
	return map[string]any{
		"type":        "number",
		"description": description,
	}
}

// PropertyBool creates a boolean property definition
func PropertyBool(description string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
	}
}

// PropertyArray creates an array property definition
func PropertyArray(description string, itemType map[string]any) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": description,
		"items":       itemType,
	}
}

// PropertyEnum creates an enum property definition
func PropertyEnum(description string, values []string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
		"enum":        values,
	}
}
