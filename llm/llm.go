// Package llm provides the main API for making LLM calls.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/i2y/bucephalus/provider"
	"github.com/i2y/bucephalus/schema"
)

// Call makes an LLM call and returns a text response.
//
// Example:
//
//	resp, err := llm.Call(ctx, "Recommend a fantasy book",
//	    llm.WithProvider("openai"),
//	    llm.WithModel("o4-mini"),
//	)
//	if err != nil {
//	    return err
//	}
//	fmt.Println(resp.Text())
func Call(ctx context.Context, prompt string, opts ...Option) (Response[string], error) {
	cfg := newCallConfig()
	cfg.apply(opts...)

	if cfg.providerName == "" {
		return Response[string]{}, ErrProviderRequired
	}
	if cfg.model == "" {
		return Response[string]{}, ErrModelRequired
	}

	p, err := provider.Get(cfg.providerName)
	if err != nil {
		return Response[string]{}, fmt.Errorf("getting provider: %w", err)
	}

	req := cfg.buildRequest(prompt)

	resp, err := p.Call(ctx, req)
	if err != nil {
		return Response[string]{}, fmt.Errorf("calling provider: %w", err)
	}

	// Build message history for Resume support
	messages := buildMessagesFromRequest(req, resp)
	config := &responseConfig{
		providerName: cfg.providerName,
		model:        cfg.model,
		tools:        cfg.tools,
	}

	return newResponseWithHistory(resp, resp.Content, nil, messages, config), nil
}

// CallParse makes an LLM call with structured output and parses the response into type T.
// The JSON schema is automatically generated from T.
//
// Example:
//
//	type Book struct {
//	    Title  string `json:"title" jsonschema:"required,description=Book title"`
//	    Author string `json:"author" jsonschema:"required"`
//	}
//
//	resp, err := llm.CallParse[Book](ctx, "Recommend a sci-fi book",
//	    llm.WithProvider("openai"),
//	    llm.WithModel("o4-mini"),
//	)
//	if err != nil {
//	    return err
//	}
//	book := resp.MustParse()
//	fmt.Printf("%s by %s\n", book.Title, book.Author)
func CallParse[T any](ctx context.Context, prompt string, opts ...Option) (Response[T], error) {
	cfg := newCallConfig()
	cfg.apply(opts...)

	if cfg.providerName == "" {
		return Response[T]{}, ErrProviderRequired
	}
	if cfg.model == "" {
		return Response[T]{}, ErrModelRequired
	}

	// Generate JSON schema from T
	jsonSchema, err := schema.Generate[T]()
	if err != nil {
		return Response[T]{}, fmt.Errorf("generating schema: %w", err)
	}

	// Get the type name for the schema
	var zero T
	typeName := reflect.TypeOf(zero).Name()
	if typeName == "" {
		typeName = "response"
	}

	cfg.jsonSchema = &provider.JSONSchema{
		Name:   typeName,
		Strict: true,
		Schema: jsonSchema,
	}

	p, err := provider.Get(cfg.providerName)
	if err != nil {
		return Response[T]{}, fmt.Errorf("getting provider: %w", err)
	}

	req := cfg.buildRequest(prompt)

	resp, err := p.Call(ctx, req)
	if err != nil {
		return Response[T]{}, fmt.Errorf("calling provider: %w", err)
	}

	// Parse the response into T
	var parsed T
	parseErr := json.Unmarshal([]byte(resp.Content), &parsed)
	if parseErr != nil {
		parseErr = &ParseError{
			Content: resp.Content,
			Target:  typeName,
			Cause:   parseErr,
		}
	}

	// Build message history for Resume support
	messages := buildMessagesFromRequest(req, resp)
	config := &responseConfig{
		providerName: cfg.providerName,
		model:        cfg.model,
		tools:        cfg.tools,
	}

	return newResponseWithHistory(resp, parsed, parseErr, messages, config), nil
}

// CallMessages makes an LLM call with a full message history.
// This is useful for multi-turn conversations.
//
// Example:
//
//	messages := []llm.Message{
//	    llm.SystemMessage("You are a helpful assistant"),
//	    llm.UserMessage("Hello"),
//	    llm.AssistantMessage("Hi! How can I help?"),
//	    llm.UserMessage("Tell me a joke"),
//	}
//
//	resp, err := llm.CallMessages(ctx, messages,
//	    llm.WithProvider("openai"),
//	    llm.WithModel("o4-mini"),
//	)
func CallMessages(ctx context.Context, messages []Message, opts ...Option) (Response[string], error) {
	cfg := newCallConfig()
	cfg.apply(opts...)

	if cfg.providerName == "" {
		return Response[string]{}, ErrProviderRequired
	}
	if cfg.model == "" {
		return Response[string]{}, ErrModelRequired
	}

	p, err := provider.Get(cfg.providerName)
	if err != nil {
		return Response[string]{}, fmt.Errorf("getting provider: %w", err)
	}

	req := cfg.buildRequestFromMessages(messages)

	resp, err := p.Call(ctx, req)
	if err != nil {
		return Response[string]{}, fmt.Errorf("calling provider: %w", err)
	}

	// Build message history for Resume support
	historyMessages := buildMessagesFromRequest(req, resp)
	config := &responseConfig{
		providerName: cfg.providerName,
		model:        cfg.model,
		tools:        cfg.tools,
	}

	return newResponseWithHistory(resp, resp.Content, nil, historyMessages, config), nil
}

// CallMessagesParse makes an LLM call with messages and parses the response.
// Combines CallMessages with structured output parsing.
func CallMessagesParse[T any](ctx context.Context, messages []Message, opts ...Option) (Response[T], error) {
	cfg := newCallConfig()
	cfg.apply(opts...)

	if cfg.providerName == "" {
		return Response[T]{}, ErrProviderRequired
	}
	if cfg.model == "" {
		return Response[T]{}, ErrModelRequired
	}

	// Generate JSON schema from T
	jsonSchema, err := schema.Generate[T]()
	if err != nil {
		return Response[T]{}, fmt.Errorf("generating schema: %w", err)
	}

	var zero T
	typeName := reflect.TypeOf(zero).Name()
	if typeName == "" {
		typeName = "response"
	}

	cfg.jsonSchema = &provider.JSONSchema{
		Name:   typeName,
		Strict: true,
		Schema: jsonSchema,
	}

	p, err := provider.Get(cfg.providerName)
	if err != nil {
		return Response[T]{}, fmt.Errorf("getting provider: %w", err)
	}

	req := cfg.buildRequestFromMessages(messages)

	resp, err := p.Call(ctx, req)
	if err != nil {
		return Response[T]{}, fmt.Errorf("calling provider: %w", err)
	}

	var parsed T
	parseErr := json.Unmarshal([]byte(resp.Content), &parsed)
	if parseErr != nil {
		parseErr = &ParseError{
			Content: resp.Content,
			Target:  typeName,
			Cause:   parseErr,
		}
	}

	// Build message history for Resume support
	historyMessages := buildMessagesFromRequest(req, resp)
	config := &responseConfig{
		providerName: cfg.providerName,
		model:        cfg.model,
		tools:        cfg.tools,
	}

	return newResponseWithHistory(resp, parsed, parseErr, historyMessages, config), nil
}

// buildMessagesFromRequest creates the full message history from request and response.
func buildMessagesFromRequest(req *provider.Request, resp *provider.Response) []Message {
	// Copy request messages
	messages := make([]Message, 0, len(req.Messages)+1)
	messages = append(messages, req.Messages...)

	// Add assistant response message
	if len(resp.ToolCalls) > 0 {
		// Convert provider.ToolCall to llm.ToolCall
		toolCalls := make([]ToolCall, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			toolCalls[i] = ToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			}
		}
		messages = append(messages, AssistantMessageWithToolCalls(resp.Content, toolCalls))
	} else {
		messages = append(messages, AssistantMessage(resp.Content))
	}

	return messages
}
