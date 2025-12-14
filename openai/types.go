package openai

import "encoding/json"

// chatCompletionRequest represents an OpenAI chat completion request.
type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []message       `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
	TopP           *float64        `json:"top_p,omitempty"`
	Seed           *int            `json:"seed,omitempty"`
	Stop           []string        `json:"stop,omitempty"`
	Tools          []toolDef       `json:"tools,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
}

// message represents a chat message.
type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// toolDef represents a tool definition.
type toolDef struct {
	Type     string      `json:"type"`
	Function functionDef `json:"function"`
}

// functionDef represents a function definition within a tool.
type functionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// responseFormat specifies the output format.
type responseFormat struct {
	Type       string            `json:"type"`
	JSONSchema *jsonSchemaFormat `json:"json_schema,omitempty"`
}

// jsonSchemaFormat specifies JSON schema for structured output.
type jsonSchemaFormat struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

// chatCompletionResponse represents an OpenAI chat completion response.
type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

// choice represents a completion choice.
type choice struct {
	Index        int             `json:"index"`
	Message      responseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// responseMessage represents the assistant's response message.
type responseMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

// toolCall represents a tool call from the assistant.
type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

// functionCall represents the function being called.
type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// usage represents token usage information.
type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// errorResponse represents an API error response.
type errorResponse struct {
	Error apiError `json:"error"`
}

// apiError represents the error details.
type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Streaming types

// streamChunk represents a streaming chunk from OpenAI.
type streamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []streamChoice `json:"choices"`
	Usage   *usage         `json:"usage,omitempty"`
}

// streamChoice represents a choice in a streaming chunk.
type streamChoice struct {
	Index        int         `json:"index"`
	Delta        streamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// streamDelta represents the delta content in a streaming chunk.
type streamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []streamToolCall `json:"tool_calls,omitempty"`
}

// streamToolCall represents a tool call delta in streaming.
type streamToolCall struct {
	Index    int                `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function streamFunctionCall `json:"function,omitempty"`
}

// streamFunctionCall represents a function call delta.
type streamFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}
