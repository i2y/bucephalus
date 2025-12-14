package provider

import "encoding/json"

// Request represents a provider-agnostic LLM request.
type Request struct {
	Model         string
	Messages      []Message
	Tools         []ToolDef
	Temperature   *float64
	MaxTokens     *int
	TopP          *float64
	TopK          *int
	Seed          *int
	StopSequences []string
	JSONSchema    *JSONSchema // For structured output
}

// Message represents a single message in the conversation.
type Message struct {
	Role      Role
	Content   string
	ToolCalls []ToolCall
	ToolID    string // When Role == RoleTool
}

// Role represents the message sender.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Response contains the LLM's response.
type Response struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason FinishReason
	Usage        Usage
}

// FinishReason indicates why the model stopped generating.
type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonToolCalls FinishReason = "tool_calls"
	FinishReasonLength    FinishReason = "length"
)

// ToolCall represents a tool invocation requested by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON string
}

// ToolDef defines a tool the model can use.
type ToolDef struct {
	Name        string
	Description string
	Parameters  json.RawMessage // JSON Schema
}

// JSONSchema represents a JSON Schema for structured output.
type JSONSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

// Usage contains token usage statistics.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
