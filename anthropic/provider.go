// Package anthropic provides an Anthropic provider implementation for Bucephalus.
package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/i2y/bucephalus/provider"
)

func init() {
	provider.Register("anthropic", func() (provider.Provider, error) {
		return New()
	})
}

// Provider implements the Anthropic Messages API.
type Provider struct {
	client *client
}

// Option configures the Anthropic provider.
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

// New creates a new Anthropic provider.
func New(opts ...Option) (*Provider, error) {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Fall back to environment variable
	if cfg.apiKey == "" {
		cfg.apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	if cfg.apiKey == "" {
		return nil, &APIError{
			Message: "Anthropic API key required: set ANTHROPIC_API_KEY or use WithAPIKey",
		}
	}

	return &Provider{
		client: newClient(cfg.apiKey, cfg.baseURL, cfg.httpClient),
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "anthropic"
}

// Call implements provider.Provider.
func (p *Provider) Call(ctx context.Context, req *provider.Request) (*provider.Response, error) {
	apiReq := p.buildRequest(req)

	apiResp, err := p.client.messages(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	return p.convertResponse(apiResp), nil
}

// CallStream implements provider.StreamingProvider.
func (p *Provider) CallStream(ctx context.Context, req *provider.Request) (provider.ResponseStream, error) {
	apiReq := p.buildRequest(req)

	stream, err := p.client.messagesStream(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	return &anthropicStream{
		reader:      stream,
		accumulated: &provider.Response{},
	}, nil
}

// buildRequest converts a provider.Request to an Anthropic API request.
func (p *Provider) buildRequest(req *provider.Request) *messagesRequest {
	apiReq := &messagesRequest{
		Model:         req.Model,
		Messages:      make([]message, 0),
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		TopK:          req.TopK,
		StopSequences: req.StopSequences,
	}

	if req.MaxTokens != nil {
		apiReq.MaxTokens = *req.MaxTokens
	}

	for _, msg := range req.Messages {
		// Extract system message
		if msg.Role == provider.RoleSystem {
			apiReq.System = msg.Content
			continue
		}

		apiMsg := message{
			Role: convertRole(msg.Role),
		}

		// Handle tool results
		if msg.Role == provider.RoleTool {
			apiMsg.Role = "user"
			apiMsg.Content = []contentPart{{
				Type:      "tool_result",
				ToolUseID: msg.ToolID,
				Content:   msg.Content,
			}}
			apiReq.Messages = append(apiReq.Messages, apiMsg)
			continue
		}

		// Handle tool calls in assistant messages
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				var input any
				if tc.Arguments != "" {
					if err := json.Unmarshal([]byte(tc.Arguments), &input); err != nil {
						// Use raw string as fallback if JSON parsing fails
						input = tc.Arguments
					}
				}
				apiMsg.Content = append(apiMsg.Content, contentPart{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				})
			}
		}

		// Add text content
		if msg.Content != "" {
			apiMsg.Content = append(apiMsg.Content, contentPart{
				Type: "text",
				Text: msg.Content,
			})
		}

		if len(apiMsg.Content) > 0 {
			apiReq.Messages = append(apiReq.Messages, apiMsg)
		}
	}

	// Handle tools
	for _, tool := range req.Tools {
		apiReq.Tools = append(apiReq.Tools, toolDef{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Parameters,
		})
	}

	// Handle JSON Schema for structured output
	if req.JSONSchema != nil {
		apiReq.OutputFormat = &outputFormat{
			Type:   "json_schema",
			Schema: req.JSONSchema.Schema,
		}
	}

	return apiReq
}

// convertResponse converts an Anthropic API response to a provider.Response.
func (p *Provider) convertResponse(resp *messagesResponse) *provider.Response {
	result := &provider.Response{
		FinishReason: convertStopReason(resp.StopReason),
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			result.Content += block.Text
		case "tool_use":
			inputJSON, _ := json.Marshal(block.Input)
			result.ToolCalls = append(result.ToolCalls, provider.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(inputJSON),
			})
		}
	}

	return result
}

func convertRole(role provider.Role) string {
	switch role {
	case provider.RoleUser:
		return "user"
	case provider.RoleAssistant:
		return "assistant"
	default:
		return string(role)
	}
}

func convertStopReason(reason string) provider.FinishReason {
	switch reason {
	case "tool_use":
		return provider.FinishReasonToolCalls
	case "max_tokens":
		return provider.FinishReasonLength
	default:
		return provider.FinishReasonStop
	}
}

// anthropicStream implements provider.ResponseStream for Anthropic.
type anthropicStream struct {
	reader      *streamReader
	accumulated *provider.Response
	err         error
	current     *provider.StreamChunk
	done        bool

	// Track current tool call for streaming
	currentToolID   string
	currentToolName string
	currentToolArgs string
}

func (s *anthropicStream) Next() bool {
	if s.done || s.err != nil {
		return false
	}

	event, err := s.reader.ReadEvent()
	if err != nil {
		if err == io.EOF {
			s.done = true
			return false
		}
		s.err = err
		return false
	}

	s.current = &provider.StreamChunk{}

	switch event.Type {
	case "content_block_start":
		if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
			s.currentToolID = event.ContentBlock.ID
			s.currentToolName = event.ContentBlock.Name
			s.currentToolArgs = ""
		}

	case "content_block_delta":
		if event.Delta != nil {
			if event.Delta.Text != "" {
				s.current.Delta = event.Delta.Text
				s.accumulated.Content += event.Delta.Text
			}
			if event.Delta.PartialJSON != "" {
				s.currentToolArgs += event.Delta.PartialJSON
				s.current.ToolCallDelta = &provider.ToolCallDelta{
					ID:             s.currentToolID,
					Name:           s.currentToolName,
					ArgumentsDelta: event.Delta.PartialJSON,
				}
			}
		}

	case "content_block_stop":
		if s.currentToolID != "" {
			s.accumulated.ToolCalls = append(s.accumulated.ToolCalls, provider.ToolCall{
				ID:        s.currentToolID,
				Name:      s.currentToolName,
				Arguments: s.currentToolArgs,
			})
			s.currentToolID = ""
			s.currentToolName = ""
			s.currentToolArgs = ""
		}

	case "message_delta":
		if event.Delta != nil && event.Delta.StopReason != "" {
			s.current.FinishReason = convertStopReason(event.Delta.StopReason)
			s.accumulated.FinishReason = s.current.FinishReason
		}
		if event.Usage != nil {
			s.accumulated.Usage.CompletionTokens = event.Usage.OutputTokens
			s.accumulated.Usage.TotalTokens = s.accumulated.Usage.PromptTokens + event.Usage.OutputTokens
		}

	case "message_start":
		if event.Message != nil {
			s.accumulated.Usage.PromptTokens = event.Message.Usage.InputTokens
		}

	case "message_stop":
		s.done = true
		return false
	}

	return true
}

func (s *anthropicStream) Current() *provider.StreamChunk {
	return s.current
}

func (s *anthropicStream) Err() error {
	return s.err
}

func (s *anthropicStream) Close() error {
	return s.reader.Close()
}

func (s *anthropicStream) Accumulated() *provider.Response {
	return s.accumulated
}
