package gemini

import "encoding/json"

// generateContentRequest represents a Gemini generateContent API request.
type generateContentRequest struct {
	Contents          []content         `json:"contents"`
	SystemInstruction *content          `json:"systemInstruction,omitempty"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
	Tools             []tool            `json:"tools,omitempty"`
}

// content represents a content object in the conversation.
type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

// part represents a part of content.
type part struct {
	Text             string            `json:"text,omitempty"`
	FunctionCall     *functionCall     `json:"functionCall,omitempty"`
	FunctionResponse *functionResponse `json:"functionResponse,omitempty"`
}

// functionCall represents a function call from the model.
type functionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// functionResponse represents a function response to send back.
type functionResponse struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

// generationConfig represents generation configuration.
type generationConfig struct {
	Temperature      *float64 `json:"temperature,omitempty"`
	MaxOutputTokens  *int     `json:"maxOutputTokens,omitempty"`
	TopP             *float64 `json:"topP,omitempty"`
	TopK             *int     `json:"topK,omitempty"`
	StopSequences    []string `json:"stopSequences,omitempty"`
	ResponseSchema   any      `json:"responseSchema,omitempty"`
	ResponseMimeType string   `json:"responseMimeType,omitempty"`
}

// tool represents a tool definition.
type tool struct {
	FunctionDeclarations []functionDeclaration `json:"functionDeclarations,omitempty"`
}

// functionDeclaration represents a function declaration.
type functionDeclaration struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// generateContentResponse represents a Gemini generateContent API response.
type generateContentResponse struct {
	Candidates    []candidate    `json:"candidates,omitempty"`
	UsageMetadata *usageMetadata `json:"usageMetadata,omitempty"`
}

// candidate represents a response candidate.
type candidate struct {
	Content       *content `json:"content,omitempty"`
	FinishReason  string   `json:"finishReason,omitempty"`
	Index         int      `json:"index,omitempty"`
	SafetyRatings []any    `json:"safetyRatings,omitempty"`
}

// usageMetadata represents token usage information.
type usageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}

// Streaming types

// streamChunk represents a chunk in the streaming response.
type streamChunk struct {
	Candidates    []candidate    `json:"candidates,omitempty"`
	UsageMetadata *usageMetadata `json:"usageMetadata,omitempty"`
}

// Error types

// errorResponse represents an API error response.
type errorResponse struct {
	Error apiError `json:"error"`
}

// apiError represents the error details.
type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}
