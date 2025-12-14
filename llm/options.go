package llm

import (
	"encoding/json"

	"github.com/i2y/bucephalus/provider"
)

// Option configures an LLM call.
type Option func(*callConfig)

// callConfig holds all configuration for a call.
type callConfig struct {
	providerName  string
	model         string
	temperature   *float64
	maxTokens     *int
	topP          *float64
	topK          *int
	seed          *int
	stopSequences []string
	systemMessage string
	tools         []Tool
	messages      []Message
	jsonSchema    *provider.JSONSchema
}

func newCallConfig() *callConfig {
	return &callConfig{}
}

func (c *callConfig) apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

// WithProvider sets the LLM provider (e.g., "openai", "anthropic").
func WithProvider(name string) Option {
	return func(c *callConfig) {
		c.providerName = name
	}
}

// WithModel sets the model to use (e.g., "o4-mini").
func WithModel(name string) Option {
	return func(c *callConfig) {
		c.model = name
	}
}

// WithTemperature sets the sampling temperature.
func WithTemperature(t float64) Option {
	return func(c *callConfig) {
		c.temperature = &t
	}
}

// WithMaxTokens sets the maximum tokens in the response.
func WithMaxTokens(n int) Option {
	return func(c *callConfig) {
		c.maxTokens = &n
	}
}

// WithTopP sets the nucleus sampling parameter (0.0 to 1.0).
// Tokens are selected from the most to least probable until the sum
// of their probabilities equals this value.
func WithTopP(p float64) Option {
	return func(c *callConfig) {
		c.topP = &p
	}
}

// WithTopK limits token selection to the k most probable tokens.
// Note: Not supported by OpenAI.
func WithTopK(k int) Option {
	return func(c *callConfig) {
		c.topK = &k
	}
}

// WithSeed sets a random seed for reproducibility.
// Note: Not supported by Anthropic.
func WithSeed(seed int) Option {
	return func(c *callConfig) {
		c.seed = &seed
	}
}

// WithStopSequences sets stop sequences to end generation.
// The model will stop generating text if one of these strings is encountered.
func WithStopSequences(seqs ...string) Option {
	return func(c *callConfig) {
		c.stopSequences = seqs
	}
}

// WithSystemMessage sets a system message.
func WithSystemMessage(msg string) Option {
	return func(c *callConfig) {
		c.systemMessage = msg
	}
}

// WithTools adds tools the model can use.
func WithTools(tools ...Tool) Option {
	return func(c *callConfig) {
		c.tools = append(c.tools, tools...)
	}
}

// WithMessages sets the conversation history.
// This is useful for multi-turn conversations with Call.
func WithMessages(msgs ...Message) Option {
	return func(c *callConfig) {
		c.messages = append(c.messages, msgs...)
	}
}

// buildRequest creates a provider.Request from the config and prompt.
func (c *callConfig) buildRequest(prompt string) *provider.Request {
	req := &provider.Request{
		Model:         c.model,
		Temperature:   c.temperature,
		MaxTokens:     c.maxTokens,
		TopP:          c.topP,
		TopK:          c.topK,
		Seed:          c.seed,
		StopSequences: c.stopSequences,
		JSONSchema:    c.jsonSchema,
	}

	// Add system message if present
	if c.systemMessage != "" {
		req.Messages = append(req.Messages, provider.Message{
			Role:    provider.RoleSystem,
			Content: c.systemMessage,
		})
	}

	// Add conversation history
	req.Messages = append(req.Messages, c.messages...)

	// Add the user prompt
	if prompt != "" {
		req.Messages = append(req.Messages, provider.Message{
			Role:    provider.RoleUser,
			Content: prompt,
		})
	}

	// Add tools
	for _, tool := range c.tools {
		params, _ := json.Marshal(tool.Parameters())
		req.Tools = append(req.Tools, provider.ToolDef{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  params,
		})
	}

	return req
}

// buildRequestFromMessages creates a provider.Request from messages.
func (c *callConfig) buildRequestFromMessages(messages []Message) *provider.Request {
	req := &provider.Request{
		Model:         c.model,
		Temperature:   c.temperature,
		MaxTokens:     c.maxTokens,
		TopP:          c.topP,
		TopK:          c.topK,
		Seed:          c.seed,
		StopSequences: c.stopSequences,
		JSONSchema:    c.jsonSchema,
		Messages:      messages,
	}

	// Add tools
	for _, tool := range c.tools {
		params, _ := json.Marshal(tool.Parameters())
		req.Tools = append(req.Tools, provider.ToolDef{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  params,
		})
	}

	return req
}
