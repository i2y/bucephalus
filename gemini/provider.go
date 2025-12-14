// Package gemini provides a Google Gemini provider implementation for Bucephalus.
package gemini

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/i2y/bucephalus/provider"
)

func init() {
	provider.Register("gemini", func() (provider.Provider, error) {
		return New()
	})
}

// Provider implements the Gemini API.
type Provider struct {
	client *client
}

// Option configures the Gemini provider.
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

// New creates a new Gemini provider.
func New(opts ...Option) (*Provider, error) {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Fall back to environment variable
	if cfg.apiKey == "" {
		cfg.apiKey = os.Getenv("GEMINI_API_KEY")
	}

	if cfg.apiKey == "" {
		return nil, &APIError{
			Message: "Gemini API key required: set GEMINI_API_KEY or use WithAPIKey",
		}
	}

	return &Provider{
		client: newClient(cfg.apiKey, cfg.baseURL, cfg.httpClient),
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "gemini"
}

// Call implements provider.Provider.
func (p *Provider) Call(ctx context.Context, req *provider.Request) (*provider.Response, error) {
	apiReq := p.buildRequest(req)

	apiResp, err := p.client.generateContent(ctx, req.Model, apiReq)
	if err != nil {
		return nil, err
	}

	return p.convertResponse(apiResp), nil
}

// CallStream implements provider.StreamingProvider.
func (p *Provider) CallStream(ctx context.Context, req *provider.Request) (provider.ResponseStream, error) {
	apiReq := p.buildRequest(req)

	stream, err := p.client.streamGenerateContent(ctx, req.Model, apiReq)
	if err != nil {
		return nil, err
	}

	return &geminiStream{
		reader:      stream,
		accumulated: &provider.Response{},
	}, nil
}

// buildRequest converts a provider.Request to a Gemini API request.
func (p *Provider) buildRequest(req *provider.Request) *generateContentRequest {
	apiReq := &generateContentRequest{
		Contents: make([]content, 0),
	}

	// Set generation config if any parameters are specified
	if req.Temperature != nil || req.MaxTokens != nil || req.TopP != nil || req.TopK != nil || len(req.StopSequences) > 0 {
		apiReq.GenerationConfig = &generationConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
			TopP:            req.TopP,
			TopK:            req.TopK,
			StopSequences:   req.StopSequences,
		}
	}

	for _, msg := range req.Messages {
		// Extract system message
		if msg.Role == provider.RoleSystem {
			apiReq.SystemInstruction = &content{
				Parts: []part{{Text: msg.Content}},
			}
			continue
		}

		apiContent := content{
			Role:  convertRole(msg.Role),
			Parts: make([]part, 0),
		}

		// Handle tool results
		if msg.Role == provider.RoleTool {
			// In Gemini, function responses go in "user" role
			// but with functionResponse part
			var responseData any
			_ = json.Unmarshal([]byte(msg.Content), &responseData)
			if responseData == nil {
				responseData = msg.Content
			}

			apiContent.Role = "user"
			apiContent.Parts = append(apiContent.Parts, part{
				FunctionResponse: &functionResponse{
					Name:     msg.ToolID,
					Response: responseData,
				},
			})
			apiReq.Contents = append(apiReq.Contents, apiContent)
			continue
		}

		// Handle tool calls in assistant messages
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				if tc.Arguments != "" {
					if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
						// Initialize empty map if JSON parsing fails
						args = make(map[string]any)
					}
				}
				apiContent.Parts = append(apiContent.Parts, part{
					FunctionCall: &functionCall{
						Name: tc.Name,
						Args: args,
					},
				})
			}
		}

		// Add text content
		if msg.Content != "" {
			apiContent.Parts = append(apiContent.Parts, part{
				Text: msg.Content,
			})
		}

		if len(apiContent.Parts) > 0 {
			apiReq.Contents = append(apiReq.Contents, apiContent)
		}
	}

	// Handle tools
	if len(req.Tools) > 0 {
		funcDecls := make([]functionDeclaration, 0, len(req.Tools))
		for _, tool := range req.Tools {
			funcDecls = append(funcDecls, functionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			})
		}
		apiReq.Tools = []tool{{FunctionDeclarations: funcDecls}}
	}

	// Handle JSON Schema for structured output
	if req.JSONSchema != nil {
		if apiReq.GenerationConfig == nil {
			apiReq.GenerationConfig = &generationConfig{}
		}
		apiReq.GenerationConfig.ResponseMimeType = "application/json"
		var schema any
		// Schema is json.RawMessage (pre-validated JSON), so unmarshal should not fail
		if err := json.Unmarshal(req.JSONSchema.Schema, &schema); err == nil {
			apiReq.GenerationConfig.ResponseSchema = schema
		}
	}

	return apiReq
}

// convertResponse converts a Gemini API response to a provider.Response.
func (p *Provider) convertResponse(resp *generateContentResponse) *provider.Response {
	result := &provider.Response{}

	if resp.UsageMetadata != nil {
		result.Usage = provider.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	if len(resp.Candidates) == 0 {
		return result
	}

	candidate := resp.Candidates[0]
	result.FinishReason = convertFinishReason(candidate.FinishReason)

	if candidate.Content != nil {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				result.Content += part.Text
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				result.ToolCalls = append(result.ToolCalls, provider.ToolCall{
					ID:        part.FunctionCall.Name, // Gemini uses name as ID
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				})
			}
		}
	}

	return result
}

func convertRole(role provider.Role) string {
	switch role {
	case provider.RoleUser:
		return "user"
	case provider.RoleAssistant:
		return "model"
	default:
		return string(role)
	}
}

func convertFinishReason(reason string) provider.FinishReason {
	switch reason {
	case "STOP":
		return provider.FinishReasonStop
	case "MAX_TOKENS":
		return provider.FinishReasonLength
	case "TOOL_USE", "FUNCTION_CALL":
		return provider.FinishReasonToolCalls
	default:
		return provider.FinishReasonStop
	}
}

// geminiStream implements provider.ResponseStream for Gemini.
type geminiStream struct {
	reader      *streamReader
	accumulated *provider.Response
	err         error
	current     *provider.StreamChunk
	done        bool
}

func (s *geminiStream) Next() bool {
	if s.done || s.err != nil {
		return false
	}

	chunk, err := s.reader.ReadChunk()
	if err != nil {
		if err == io.EOF {
			s.done = true
			return false
		}
		s.err = err
		return false
	}

	s.current = &provider.StreamChunk{}

	if chunk.UsageMetadata != nil {
		s.accumulated.Usage = provider.Usage{
			PromptTokens:     chunk.UsageMetadata.PromptTokenCount,
			CompletionTokens: chunk.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      chunk.UsageMetadata.TotalTokenCount,
		}
	}

	if len(chunk.Candidates) > 0 {
		candidate := chunk.Candidates[0]
		s.current.FinishReason = convertFinishReason(candidate.FinishReason)
		s.accumulated.FinishReason = s.current.FinishReason

		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					s.current.Delta = part.Text
					s.accumulated.Content += part.Text
				}
				if part.FunctionCall != nil {
					argsJSON, _ := json.Marshal(part.FunctionCall.Args)
					s.current.ToolCallDelta = &provider.ToolCallDelta{
						ID:             part.FunctionCall.Name,
						Name:           part.FunctionCall.Name,
						ArgumentsDelta: string(argsJSON),
					}
					s.accumulated.ToolCalls = append(s.accumulated.ToolCalls, provider.ToolCall{
						ID:        part.FunctionCall.Name,
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					})
				}
			}
		}
	}

	return true
}

func (s *geminiStream) Current() *provider.StreamChunk {
	return s.current
}

func (s *geminiStream) Err() error {
	return s.err
}

func (s *geminiStream) Close() error {
	return s.reader.Close()
}

func (s *geminiStream) Accumulated() *provider.Response {
	return s.accumulated
}
