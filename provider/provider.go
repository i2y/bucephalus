// Package provider defines the interface for LLM providers.
package provider

import "context"

// Provider is the core abstraction for LLM providers.
// All provider implementations must satisfy this interface.
type Provider interface {
	// Name returns the provider identifier (e.g., "openai", "anthropic").
	Name() string

	// Call executes a non-streaming LLM request.
	Call(ctx context.Context, req *Request) (*Response, error)
}

// StreamingProvider extends Provider with streaming capability.
type StreamingProvider interface {
	Provider

	// CallStream executes a streaming LLM request.
	CallStream(ctx context.Context, req *Request) (ResponseStream, error)
}

// ResponseStream represents a streaming response.
type ResponseStream interface {
	// Next advances to the next chunk, returns false when done.
	Next() bool

	// Current returns the current chunk.
	Current() *StreamChunk

	// Err returns any error that occurred during streaming.
	Err() error

	// Close releases stream resources.
	Close() error

	// Accumulated returns the full response accumulated so far.
	Accumulated() *Response
}

// StreamChunk represents a single streaming chunk.
type StreamChunk struct {
	Delta         string
	ToolCallDelta *ToolCallDelta
	FinishReason  FinishReason
}

// ToolCallDelta represents incremental tool call data in streaming.
type ToolCallDelta struct {
	ID             string
	Name           string
	ArgumentsDelta string
}
