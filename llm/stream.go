package llm

import (
	"context"
	"fmt"
	"iter"

	"github.com/i2y/bucephalus/provider"
)

// Stream represents a streaming response from an LLM.
type Stream struct {
	stream provider.ResponseStream
	err    error
}

// Chunks returns an iterator over the stream chunks.
// This uses Go 1.23+ range-over-func.
//
// Example:
//
//	stream, err := llm.CallStream(ctx, "Write a story", opts...)
//	if err != nil {
//	    return err
//	}
//	defer stream.Close()
//
//	for chunk := range stream.Chunks() {
//	    fmt.Print(chunk.Delta)
//	}
func (s *Stream) Chunks() iter.Seq[StreamChunk] {
	return func(yield func(StreamChunk) bool) {
		for s.stream.Next() {
			current := s.stream.Current()
			chunk := StreamChunk{
				Delta:        current.Delta,
				FinishReason: FinishReason(current.FinishReason),
			}
			if current.ToolCallDelta != nil {
				chunk.ToolCallDelta = &ToolCallDelta{
					ID:             current.ToolCallDelta.ID,
					Name:           current.ToolCallDelta.Name,
					ArgumentsDelta: current.ToolCallDelta.ArgumentsDelta,
				}
			}
			if !yield(chunk) {
				return
			}
		}
		s.err = s.stream.Err()
	}
}

// Err returns any error that occurred during streaming.
func (s *Stream) Err() error {
	return s.err
}

// Close closes the stream and releases resources.
func (s *Stream) Close() error {
	return s.stream.Close()
}

// Response returns the accumulated response after streaming is complete.
// Should be called after iterating through all chunks.
func (s *Stream) Response() Response[string] {
	accumulated := s.stream.Accumulated()
	return newParsedResponse(accumulated, accumulated.Content, nil)
}

// StreamChunk represents a single chunk in a streaming response.
type StreamChunk struct {
	Delta         string
	ToolCallDelta *ToolCallDelta
	FinishReason  FinishReason
}

// ToolCallDelta represents incremental tool call data.
type ToolCallDelta struct {
	ID             string
	Name           string
	ArgumentsDelta string
}

// CallStream makes a streaming LLM call.
//
// Example:
//
//	stream, err := llm.CallStream(ctx, "Write a short story",
//	    llm.WithProvider("openai"),
//	    llm.WithModel("o4-mini"),
//	)
//	if err != nil {
//	    return err
//	}
//	defer stream.Close()
//
//	for chunk := range stream.Chunks() {
//	    fmt.Print(chunk.Delta)
//	}
//
//	if err := stream.Err(); err != nil {
//	    return err
//	}
func CallStream(ctx context.Context, prompt string, opts ...Option) (*Stream, error) {
	cfg := newCallConfig()
	cfg.apply(opts...)

	if cfg.providerName == "" {
		return nil, ErrProviderRequired
	}
	if cfg.model == "" {
		return nil, ErrModelRequired
	}

	p, err := provider.Get(cfg.providerName)
	if err != nil {
		return nil, fmt.Errorf("getting provider: %w", err)
	}

	// Check if provider supports streaming
	sp, ok := p.(provider.StreamingProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not support streaming", cfg.providerName)
	}

	req := cfg.buildRequest(prompt)

	stream, err := sp.CallStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("starting stream: %w", err)
	}

	return &Stream{stream: stream}, nil
}

// CallMessagesStream makes a streaming LLM call with message history.
func CallMessagesStream(ctx context.Context, messages []Message, opts ...Option) (*Stream, error) {
	cfg := newCallConfig()
	cfg.apply(opts...)

	if cfg.providerName == "" {
		return nil, ErrProviderRequired
	}
	if cfg.model == "" {
		return nil, ErrModelRequired
	}

	p, err := provider.Get(cfg.providerName)
	if err != nil {
		return nil, fmt.Errorf("getting provider: %w", err)
	}

	sp, ok := p.(provider.StreamingProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not support streaming", cfg.providerName)
	}

	req := cfg.buildRequestFromMessages(messages)

	stream, err := sp.CallStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("starting stream: %w", err)
	}

	return &Stream{stream: stream}, nil
}
