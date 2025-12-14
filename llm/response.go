package llm

import (
	"context"
	"fmt"

	"github.com/i2y/bucephalus/provider"
)

// Response wraps the provider response with type-safe parsed content.
// T is the type of structured output expected from the LLM.
type Response[T any] struct {
	raw       *provider.Response
	parsed    T
	hasParsed bool
	parseErr  error
	messages  []Message       // Full conversation history
	config    *responseConfig // Provider/model info for Resume
}

// responseConfig stores the configuration needed to resume a conversation.
type responseConfig struct {
	providerName string
	model        string
	tools        []Tool
}

// Text returns the raw text content of the response.
func (r Response[T]) Text() string {
	if r.raw == nil {
		return ""
	}
	return r.raw.Content
}

// Parsed returns the structured output with compile-time type safety.
// Returns ErrNotParsed if the response was not created via CallParse.
func (r Response[T]) Parsed() (T, error) {
	if r.parseErr != nil {
		return r.parsed, r.parseErr
	}
	if !r.hasParsed {
		return r.parsed, ErrNotParsed
	}
	return r.parsed, nil
}

// MustParse returns the parsed value or panics.
// Useful in tests or when you're certain parsing succeeded.
func (r Response[T]) MustParse() T {
	v, err := r.Parsed()
	if err != nil {
		panic(err)
	}
	return v
}

// HasToolCalls returns true if the response contains tool calls.
func (r Response[T]) HasToolCalls() bool {
	return r.raw != nil && len(r.raw.ToolCalls) > 0
}

// ToolCalls returns any tool calls made by the model.
func (r Response[T]) ToolCalls() []ToolCall {
	if r.raw == nil {
		return nil
	}
	calls := make([]ToolCall, len(r.raw.ToolCalls))
	for i, tc := range r.raw.ToolCalls {
		calls[i] = ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
		}
	}
	return calls
}

// Usage returns token usage statistics.
func (r Response[T]) Usage() Usage {
	if r.raw == nil {
		return Usage{}
	}
	return Usage{
		PromptTokens:     r.raw.Usage.PromptTokens,
		CompletionTokens: r.raw.Usage.CompletionTokens,
		TotalTokens:      r.raw.Usage.TotalTokens,
	}
}

// FinishReason returns why the model stopped generating.
func (r Response[T]) FinishReason() FinishReason {
	if r.raw == nil {
		return ""
	}
	return FinishReason(r.raw.FinishReason)
}

// Raw returns the underlying provider response.
// This can be useful for debugging or accessing provider-specific data.
func (r Response[T]) Raw() *provider.Response {
	return r.raw
}

// Messages returns the full conversation history including the assistant's response.
func (r Response[T]) Messages() []Message {
	return r.messages
}

// Resume continues the conversation with additional user content.
// It uses the same provider, model, and tools from the original call.
//
// Example:
//
//	resp, _ := llm.Call(ctx, "Recommend a book", opts...)
//	continuation, _ := resp.Resume(ctx, "Why did you recommend that one?")
//	fmt.Println(continuation.Text())
func (r Response[T]) Resume(ctx context.Context, content string, opts ...Option) (Response[string], error) {
	if r.config == nil {
		return Response[string]{}, fmt.Errorf("cannot resume: response was not created with Resume support")
	}

	// Build new messages with the user's continuation
	newMessages := make([]Message, len(r.messages), len(r.messages)+1)
	copy(newMessages, r.messages)
	newMessages = append(newMessages, UserMessage(content))

	// Build options: start with original config, then apply any overrides
	allOpts := make([]Option, 0, len(opts)+3)
	allOpts = append(allOpts, WithProvider(r.config.providerName), WithModel(r.config.model))
	if len(r.config.tools) > 0 {
		allOpts = append(allOpts, WithTools(r.config.tools...))
	}
	allOpts = append(allOpts, opts...)

	return CallMessages(ctx, newMessages, allOpts...)
}

// ResumeWithToolOutputs continues the conversation with tool execution results.
// This is used after the LLM has requested tool calls.
//
// Example:
//
//	if resp.HasToolCalls() {
//	    toolMessages, _ := llm.ExecuteToolCalls(ctx, resp.ToolCalls(), registry)
//	    continuation, _ := resp.ResumeWithToolOutputs(ctx, toolMessages)
//	    fmt.Println(continuation.Text())
//	}
func (r Response[T]) ResumeWithToolOutputs(ctx context.Context, toolOutputs []Message, opts ...Option) (Response[string], error) {
	if r.config == nil {
		return Response[string]{}, fmt.Errorf("cannot resume: response was not created with Resume support")
	}

	// Build new messages with tool outputs
	newMessages := make([]Message, len(r.messages), len(r.messages)+len(toolOutputs))
	copy(newMessages, r.messages)
	newMessages = append(newMessages, toolOutputs...)

	// Build options: start with original config, then apply any overrides
	allOpts := make([]Option, 0, len(opts)+3)
	allOpts = append(allOpts, WithProvider(r.config.providerName), WithModel(r.config.model))
	if len(r.config.tools) > 0 {
		allOpts = append(allOpts, WithTools(r.config.tools...))
	}
	allOpts = append(allOpts, opts...)

	return CallMessages(ctx, newMessages, allOpts...)
}

// Usage contains token usage information.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ToolCall represents a tool call from the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON string
}

// FinishReason indicates why the model stopped generating.
type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonToolCalls FinishReason = "tool_calls"
	FinishReasonLength    FinishReason = "length"
)

// newParsedResponse creates a Response with parsed content.
func newParsedResponse[T any](raw *provider.Response, parsed T, parseErr error) Response[T] {
	return Response[T]{
		raw:       raw,
		parsed:    parsed,
		hasParsed: parseErr == nil,
		parseErr:  parseErr,
	}
}

// newResponseWithHistory creates a Response with conversation history and config for Resume support.
func newResponseWithHistory[T any](raw *provider.Response, parsed T, parseErr error, messages []Message, config *responseConfig) Response[T] {
	return Response[T]{
		raw:       raw,
		parsed:    parsed,
		hasParsed: parseErr == nil,
		parseErr:  parseErr,
		messages:  messages,
		config:    config,
	}
}
