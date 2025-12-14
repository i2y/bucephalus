package anthropic

import "encoding/json"

// messagesRequest represents an Anthropic Messages API request.
type messagesRequest struct {
	Model         string        `json:"model"`
	Messages      []message     `json:"messages"`
	System        string        `json:"system,omitempty"`
	MaxTokens     int           `json:"max_tokens"`
	Temperature   *float64      `json:"temperature,omitempty"`
	TopP          *float64      `json:"top_p,omitempty"`
	TopK          *int          `json:"top_k,omitempty"`
	StopSequences []string      `json:"stop_sequences,omitempty"`
	Tools         []toolDef     `json:"tools,omitempty"`
	Stream        bool          `json:"stream,omitempty"`
	OutputFormat  *outputFormat `json:"output_format,omitempty"`
}

// outputFormat specifies the output format for structured output.
type outputFormat struct {
	Type   string          `json:"type"`   // "json_schema"
	Schema json.RawMessage `json:"schema"` // The JSON schema
}

// message represents a message in the conversation.
type message struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

// contentPart represents a part of message content.
type contentPart struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"` // For tool_result
}

// toolDef represents a tool definition.
type toolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// messagesResponse represents an Anthropic Messages API response.
type messagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []contentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        messagesUsage  `json:"usage"`
}

// contentBlock represents a content block in the response.
type contentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

// messagesUsage represents token usage information.
type messagesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Streaming event types
type streamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta *delta `json:"delta,omitempty"`
	// For message_start
	Message *messagesResponse `json:"message,omitempty"`
	// For content_block_start
	ContentBlock *contentBlock `json:"content_block,omitempty"`
	// For message_delta
	Usage *deltaUsage `json:"usage,omitempty"`
}

type delta struct {
	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
}

type deltaUsage struct {
	OutputTokens int `json:"output_tokens"`
}

// errorResponse represents an API error response.
type errorResponse struct {
	Type  string   `json:"type"`
	Error apiError `json:"error"`
}

// apiError represents the error details.
type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
