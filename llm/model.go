package llm

import (
	"context"
	"encoding/json"

	"github.com/i2y/bucephalus/provider"
	"github.com/i2y/bucephalus/schema"
)

// Model represents a configured LLM model with default options.
// It provides a convenient way to reuse common configuration.
//
// Example:
//
//	model := llm.NewModel("openai", "gpt-4o-mini",
//	    llm.WithTemperature(0.7),
//	)
//
//	resp, err := model.Call(ctx, "Tell me a joke")
type Model struct {
	providerName string
	modelName    string
	baseOpts     []Option
}

// NewModel creates a new Model with the given provider and model name.
// Additional options can be provided as default configuration.
func NewModel(providerName, modelName string, opts ...Option) *Model {
	return &Model{
		providerName: providerName,
		modelName:    modelName,
		baseOpts:     opts,
	}
}

// Call makes an LLM call using this model's configuration.
// Per-call options override the model's base options.
func (m *Model) Call(ctx context.Context, prompt string, opts ...Option) (Response[string], error) {
	allOpts := m.mergeOptions(opts)
	return Call(ctx, prompt, allOpts...)
}

// CallParse makes an LLM call with structured output using this model.
// Per-call options override the model's base options.
func (m *Model) CallParse(ctx context.Context, prompt string, target any, opts ...Option) error {
	allOpts := m.mergeOptions(opts)
	// Use reflection-based approach for non-generic method
	return callParseReflect(ctx, prompt, target, allOpts...)
}

// CallMessages makes an LLM call with message history using this model.
func (m *Model) CallMessages(ctx context.Context, messages []Message, opts ...Option) (Response[string], error) {
	allOpts := m.mergeOptions(opts)
	return CallMessages(ctx, messages, allOpts...)
}

// mergeOptions combines base options with per-call options.
func (m *Model) mergeOptions(opts []Option) []Option {
	allOpts := make([]Option, 0, len(m.baseOpts)+len(opts)+2)
	allOpts = append(allOpts, WithProvider(m.providerName), WithModel(m.modelName))
	allOpts = append(allOpts, m.baseOpts...)
	allOpts = append(allOpts, opts...) // Per-call opts override base opts
	return allOpts
}

// callParseReflect is a helper for Model.CallParse that uses reflection.
// This is necessary because Model.CallParse cannot be generic.
func callParseReflect(ctx context.Context, prompt string, target any, opts ...Option) error {
	cfg := newCallConfig()
	cfg.apply(opts...)

	if cfg.providerName == "" {
		return ErrProviderRequired
	}
	if cfg.model == "" {
		return ErrModelRequired
	}

	// Generate schema from target
	jsonSchema, err := schema.GenerateFromValue(target)
	if err != nil {
		return err
	}

	cfg.jsonSchema = &provider.JSONSchema{
		Name:   "response",
		Strict: true,
		Schema: jsonSchema,
	}

	p, err := provider.Get(cfg.providerName)
	if err != nil {
		return err
	}

	req := cfg.buildRequest(prompt)

	resp, err := p.Call(ctx, req)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(resp.Content), target)
}
