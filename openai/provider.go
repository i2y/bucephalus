// Package openai provides an OpenAI provider implementation for Bucephalus.
package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/i2y/bucephalus/provider"
)

func init() {
	provider.Register("openai", func() (provider.Provider, error) {
		return New()
	})
}

// Provider implements the OpenAI API.
type Provider struct {
	client *client
}

// Option configures the OpenAI provider.
type Option func(*providerConfig)

type providerConfig struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// WithAPIKey sets the API key.
func WithAPIKey(key string) Option {
	return func(c *providerConfig) {
		c.apiKey = key
	}
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) Option {
	return func(c *providerConfig) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *providerConfig) {
		c.httpClient = client
	}
}

// New creates a new OpenAI provider.
func New(opts ...Option) (*Provider, error) {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Fall back to environment variable
	if cfg.apiKey == "" {
		cfg.apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if cfg.apiKey == "" {
		return nil, &APIError{
			Message: "OpenAI API key required: set OPENAI_API_KEY or use WithAPIKey",
		}
	}

	return &Provider{
		client: newClient(cfg.apiKey, cfg.baseURL, cfg.httpClient),
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "openai"
}

// Call implements provider.Provider.
func (p *Provider) Call(ctx context.Context, req *provider.Request) (*provider.Response, error) {
	apiReq := p.buildRequest(req)

	apiResp, err := p.client.chatCompletion(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	return p.convertResponse(apiResp), nil
}

// CallStream implements provider.StreamingProvider.
func (p *Provider) CallStream(ctx context.Context, req *provider.Request) (provider.ResponseStream, error) {
	apiReq := p.buildRequest(req)

	stream, err := p.client.chatCompletionStream(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	return &openaiStream{
		reader:      stream,
		accumulated: &provider.Response{},
		toolCalls:   make(map[int]*provider.ToolCall),
	}, nil
}

// buildRequest converts a provider.Request to an OpenAI API request.
func (p *Provider) buildRequest(req *provider.Request) *chatCompletionRequest {
	apiReq := &chatCompletionRequest{
		Model:       req.Model,
		Messages:    make([]message, 0, len(req.Messages)),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Seed:        req.Seed,
		Stop:        req.StopSequences,
	}

	for _, msg := range req.Messages {
		apiMsg := message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		// Handle tool call ID for tool results
		if msg.ToolID != "" {
			apiMsg.ToolCallID = msg.ToolID
		}

		// Handle tool calls in assistant messages
		if len(msg.ToolCalls) > 0 {
			apiMsg.ToolCalls = make([]toolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				apiMsg.ToolCalls[i] = toolCall{
					ID:   tc.ID,
					Type: "function",
					Function: functionCall{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}
		}

		apiReq.Messages = append(apiReq.Messages, apiMsg)
	}

	// Handle tools
	for _, tool := range req.Tools {
		apiReq.Tools = append(apiReq.Tools, toolDef{
			Type: "function",
			Function: functionDef{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}

	// Handle JSON Schema for structured output
	if req.JSONSchema != nil {
		apiReq.ResponseFormat = &responseFormat{
			Type: "json_schema",
			JSONSchema: &jsonSchemaFormat{
				Name:   req.JSONSchema.Name,
				Strict: req.JSONSchema.Strict,
				Schema: makeAllPropertiesRequired(req.JSONSchema.Schema),
			},
		}
	}

	return apiReq
}

// convertResponse converts an OpenAI API response to a provider.Response.
func (p *Provider) convertResponse(resp *chatCompletionResponse) *provider.Response {
	if len(resp.Choices) == 0 {
		return &provider.Response{}
	}

	choice := resp.Choices[0]
	result := &provider.Response{
		Content:      choice.Message.Content,
		FinishReason: convertFinishReason(choice.FinishReason),
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Convert tool calls
	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, provider.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return result
}

// makeAllPropertiesRequired ensures all properties in the schema are required.
// OpenAI's structured output API requires all properties to be in the 'required' array.
func makeAllPropertiesRequired(schema json.RawMessage) json.RawMessage {
	if schema == nil {
		return nil
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		return schema
	}

	makeRequiredRecursive(schemaMap)

	result, err := json.Marshal(schemaMap)
	if err != nil {
		return schema
	}
	return result
}

// makeRequiredRecursive recursively makes all properties required in the schema.
func makeRequiredRecursive(schemaMap map[string]any) {
	// Get all property names and make them required
	if props, ok := schemaMap["properties"].(map[string]any); ok {
		required := make([]string, 0, len(props))
		for key := range props {
			required = append(required, key)
		}
		schemaMap["required"] = required

		// Recursively process nested objects
		for _, val := range props {
			if propMap, ok := val.(map[string]any); ok {
				// Handle nested object types
				if propMap["type"] == "object" {
					makeRequiredRecursive(propMap)
				}
				// Handle array items
				if items, ok := propMap["items"].(map[string]any); ok {
					if items["type"] == "object" {
						makeRequiredRecursive(items)
					}
				}
			}
		}
	}
}

// convertFinishReason converts an OpenAI finish reason to a provider.FinishReason.
func convertFinishReason(reason string) provider.FinishReason {
	switch reason {
	case "tool_calls":
		return provider.FinishReasonToolCalls
	case "length":
		return provider.FinishReasonLength
	default:
		return provider.FinishReasonStop
	}
}

// openaiStream implements provider.ResponseStream for OpenAI.
type openaiStream struct {
	reader      *streamReader
	accumulated *provider.Response
	err         error
	current     *provider.StreamChunk
	done        bool
	toolCalls   map[int]*provider.ToolCall // Track tool calls by index
}

func (s *openaiStream) Next() bool {
	if s.done || s.err != nil {
		return false
	}

	chunk, err := s.reader.ReadChunk()
	if err != nil {
		if err.Error() == "EOF" {
			s.done = true
			// Finalize tool calls
			for _, tc := range s.toolCalls {
				s.accumulated.ToolCalls = append(s.accumulated.ToolCalls, *tc)
			}
			return false
		}
		s.err = err
		return false
	}

	s.current = &provider.StreamChunk{}

	if len(chunk.Choices) > 0 {
		choice := chunk.Choices[0]
		delta := choice.Delta

		// Handle content delta
		if delta.Content != "" {
			s.current.Delta = delta.Content
			s.accumulated.Content += delta.Content
		}

		// Handle tool call deltas
		for _, tc := range delta.ToolCalls {
			if _, exists := s.toolCalls[tc.Index]; !exists {
				s.toolCalls[tc.Index] = &provider.ToolCall{}
			}
			toolCall := s.toolCalls[tc.Index]

			if tc.ID != "" {
				toolCall.ID = tc.ID
			}
			if tc.Function.Name != "" {
				toolCall.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				toolCall.Arguments += tc.Function.Arguments
				s.current.ToolCallDelta = &provider.ToolCallDelta{
					ID:             toolCall.ID,
					Name:           toolCall.Name,
					ArgumentsDelta: tc.Function.Arguments,
				}
			}
		}

		// Handle finish reason
		if choice.FinishReason != nil {
			s.current.FinishReason = convertFinishReason(*choice.FinishReason)
			s.accumulated.FinishReason = s.current.FinishReason
		}
	}

	// Handle usage (sent in final chunk with stream_options)
	if chunk.Usage != nil {
		s.accumulated.Usage = provider.Usage{
			PromptTokens:     chunk.Usage.PromptTokens,
			CompletionTokens: chunk.Usage.CompletionTokens,
			TotalTokens:      chunk.Usage.TotalTokens,
		}
	}

	return true
}

func (s *openaiStream) Current() *provider.StreamChunk {
	return s.current
}

func (s *openaiStream) Err() error {
	return s.err
}

func (s *openaiStream) Close() error {
	return s.reader.Close()
}

func (s *openaiStream) Accumulated() *provider.Response {
	return s.accumulated
}
