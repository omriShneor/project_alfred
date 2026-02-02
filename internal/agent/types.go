package agent

import "time"

// Message represents a conversation message in the Anthropic API format
type Message struct {
	Role    string         `json:"role"` // "user" or "assistant"
	Content []ContentBlock `json:"content"`
}

// ContentBlock is the interface for different content types
type ContentBlock interface {
	BlockType() string
}

// TextBlock represents plain text content
type TextBlock struct {
	Type string `json:"type"` // Always "text"
	Text string `json:"text"`
}

func (t TextBlock) BlockType() string { return "text" }

// ToolUseBlock represents a tool invocation by the assistant
type ToolUseBlock struct {
	Type  string                 `json:"type"` // Always "tool_use"
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]any `json:"input"`
}

func (t ToolUseBlock) BlockType() string { return "tool_use" }

// ToolResultBlock represents the result of a tool execution
type ToolResultBlock struct {
	Type      string `json:"type"` // Always "tool_result"
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

func (t ToolResultBlock) BlockType() string { return "tool_result" }

// AgentInput provides context for agent execution
type AgentInput struct {
	// Messages is the conversation history
	Messages []Message

	// Context provides additional data for the system prompt
	Context map[string]any

	// MaxTurns limits the number of API round-trips (default: 1 for single-shot)
	MaxTurns int
}

// AgentOutput contains the result of agent execution
type AgentOutput struct {
	// ToolCalls contains all tool calls made during execution
	ToolCalls []ToolCall

	// Conversation contains all messages exchanged
	Conversation []Message

	// Usage contains token usage statistics
	Usage UsageStats

	// FinalText contains any final text response from the agent
	FinalText string
}

// ToolCall represents a single tool invocation and its result
type ToolCall struct {
	Name   string                 `json:"name"`
	Input  map[string]any `json:"input"`
	Output string                 `json:"output"`
	Error  error                  `json:"error,omitempty"`
}

// UsageStats tracks API usage
type UsageStats struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Add accumulates usage from another stats object
func (u *UsageStats) Add(other UsageStats) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.TotalTokens += other.TotalTokens
}

// EventAnalysis represents the result of event analysis (for backward compatibility)
type EventAnalysis struct {
	HasEvent   bool       `json:"has_event"`
	Action     string     `json:"action"` // "create", "update", "delete", "none"
	Event      *EventData `json:"event,omitempty"`
	Reasoning  string     `json:"reasoning"`
	Confidence float64    `json:"confidence"`
}

// EventData contains the extracted event details
type EventData struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	StartTime      string `json:"start_time"` // ISO 8601 format
	EndTime        string `json:"end_time,omitempty"`
	Location       string `json:"location,omitempty"`
	UpdateRef      string `json:"update_ref,omitempty"`       // Google event ID for updates/deletes
	AlfredEventRef int64  `json:"alfred_event_ref,omitempty"` // Internal DB ID for pending events
}

// DateTimeResult represents extracted date/time information
type DateTimeResult struct {
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	IsAllDay    bool       `json:"is_all_day"`
	Timezone    string     `json:"timezone,omitempty"`
	Confidence  float64    `json:"confidence"`
	RawText     string     `json:"raw_text"` // Original text that was parsed
}

// LocationResult represents extracted location information
type LocationResult struct {
	Name       string  `json:"name"`
	Address    string  `json:"address,omitempty"`
	Type       string  `json:"type"` // "physical", "virtual", "unknown"
	URL        string  `json:"url,omitempty"` // For virtual meetings
	Confidence float64 `json:"confidence"`
	RawText    string  `json:"raw_text"`
}

// AttendeeResult represents extracted attendee information
type AttendeeResult struct {
	Name       string  `json:"name"`
	Email      string  `json:"email,omitempty"`
	Phone      string  `json:"phone,omitempty"`
	Role       string  `json:"role"` // "organizer", "required", "optional"
	Confidence float64 `json:"confidence"`
}

// AttendeesResult contains all extracted attendees
type AttendeesResult struct {
	Attendees []AttendeeResult `json:"attendees"`
}
