package plugin

import (
	"context"

	"github.com/i2y/bucephalus/llm"
)

// AgentRunner provides methods to run an agent as an independent LLM call.
// It maintains conversation history across multiple Run() calls via AgentContext.
type AgentRunner struct {
	agent          *Agent
	providerName   string
	model          string
	availableTools []llm.Tool
	filteredTools  []llm.Tool
	temperature    *float64
	maxTokens      *int
	context        *AgentContext // Maintains conversation history and state
	extraLLMOpts   []llm.Option  // Additional llm.Options to apply on every call
}

// AgentOption configures an AgentRunner.
type AgentOption func(*AgentRunner)

// WithAgentProvider sets the provider for the agent runner.
func WithAgentProvider(name string) AgentOption {
	return func(r *AgentRunner) {
		r.providerName = name
	}
}

// WithAgentModel sets the model for the agent runner.
func WithAgentModel(model string) AgentOption {
	return func(r *AgentRunner) {
		r.model = model
	}
}

// WithAgentTools provides tools that the agent can use.
// Only tools listed in the agent's Tools field will actually be available.
func WithAgentTools(tools ...llm.Tool) AgentOption {
	return func(r *AgentRunner) {
		r.availableTools = tools
	}
}

// WithAgentTemperature sets the temperature for the agent.
func WithAgentTemperature(t float64) AgentOption {
	return func(r *AgentRunner) {
		r.temperature = &t
	}
}

// WithAgentMaxTokens sets the max tokens for the agent.
func WithAgentMaxTokens(n int) AgentOption {
	return func(r *AgentRunner) {
		r.maxTokens = &n
	}
}

// WithAgentContext sets an existing context for the runner.
// This allows sharing context between agents or continuing from a previous state.
func WithAgentContext(ctx *AgentContext) AgentOption {
	return func(r *AgentRunner) {
		r.context = ctx
	}
}

// WithAgentLLMOptions sets additional llm.Options to apply on every Run() call.
// This allows passing options like WithTopP, WithTopK, WithSeed, WithStopSequences,
// or additional WithSystemMessage to the agent.
//
// Example:
//
//	runner := agent.NewRunner(
//	    plugin.WithAgentProvider("anthropic"),
//	    plugin.WithAgentModel("claude-3-5-haiku-latest"),
//	    plugin.WithAgentLLMOptions(
//	        llm.WithTopP(0.9),
//	        llm.WithSystemMessage(p.SkillsIndexSystemMessage()),
//	    ),
//	)
func WithAgentLLMOptions(opts ...llm.Option) AgentOption {
	return func(r *AgentRunner) {
		r.extraLLMOpts = append(r.extraLLMOpts, opts...)
	}
}

// RunOption configures a single Run() call.
type RunOption func(*runConfig)

// runConfig holds configuration for a single Run() call.
type runConfig struct {
	extraSystemMessage string
	extraLLMOpts       []llm.Option
}

// WithRunSystemMessage adds an additional system message for this Run() call only.
// This is useful for adding context-specific instructions without modifying the runner.
func WithRunSystemMessage(msg string) RunOption {
	return func(c *runConfig) {
		c.extraSystemMessage = msg
	}
}

// WithRunLLMOptions adds additional llm.Options for this Run() call only.
func WithRunLLMOptions(opts ...llm.Option) RunOption {
	return func(c *runConfig) {
		c.extraLLMOpts = append(c.extraLLMOpts, opts...)
	}
}

// NewRunner creates a new AgentRunner for this agent.
// The runner maintains conversation history across multiple Run() calls.
func (a *Agent) NewRunner(opts ...AgentOption) *AgentRunner {
	runner := &AgentRunner{
		agent: a,
	}

	for _, opt := range opts {
		opt(runner)
	}

	// Filter tools based on agent's allowed tools
	runner.filteredTools = runner.filterTools()

	// Initialize context if not provided via options
	if runner.context == nil {
		runner.context = NewAgentContext()
	}

	return runner
}

// filterTools filters available tools to only include those allowed by the agent.
func (r *AgentRunner) filterTools() []llm.Tool {
	if len(r.agent.Tools) == 0 || len(r.availableTools) == 0 {
		return nil
	}

	// Create a set of allowed tool names
	allowed := make(map[string]bool)
	for _, name := range r.agent.Tools {
		allowed[name] = true
	}

	// Filter tools
	var filtered []llm.Tool
	for _, tool := range r.availableTools {
		if allowed[tool.Name()] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// Run executes the agent with a task and returns the response.
// Conversation history is maintained in the runner's context, allowing
// multi-turn conversations across multiple Run() calls.
//
// Optional RunOption arguments can be passed to customize this specific call:
//
//	resp, _ := runner.Run(ctx, "Help me",
//	    plugin.WithRunSystemMessage("Additional context for this call"),
//	    plugin.WithRunLLMOptions(llm.WithTopP(0.9)),
//	)
func (r *AgentRunner) Run(ctx context.Context, task string, runOpts ...RunOption) (llm.Response[string], error) {
	// Apply run options
	cfg := &runConfig{}
	for _, opt := range runOpts {
		opt(cfg)
	}

	// Build options
	opts := make([]llm.Option, 0)

	if r.providerName != "" {
		opts = append(opts, llm.WithProvider(r.providerName))
	}
	if r.model != "" {
		opts = append(opts, llm.WithModel(r.model))
	}
	if r.temperature != nil {
		opts = append(opts, llm.WithTemperature(*r.temperature))
	}
	if r.maxTokens != nil {
		opts = append(opts, llm.WithMaxTokens(*r.maxTokens))
	}

	// Add agent's system message
	opts = append(opts, llm.WithSystemMessage(r.agent.ToSystemMessage()))

	// Add extra system message from run options (if any)
	if cfg.extraSystemMessage != "" {
		opts = append(opts, llm.WithSystemMessage(cfg.extraSystemMessage))
	}

	// Add filtered tools
	if len(r.filteredTools) > 0 {
		opts = append(opts, llm.WithTools(r.filteredTools...))
	}

	// Add runner-level extra LLM options
	opts = append(opts, r.extraLLMOpts...)

	// Add run-level extra LLM options
	opts = append(opts, cfg.extraLLMOpts...)

	// Create user message for this turn
	userMsg := llm.UserMessage(task)

	// Build messages: existing history + new user message
	history := r.context.History()
	messages := make([]llm.Message, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, userMsg)

	// Make the LLM call with full message history
	resp, err := llm.CallMessages(ctx, messages, opts...)
	if err != nil {
		return resp, err
	}

	// Add user message and assistant response to context history
	r.context.AddMessage(userMsg)
	r.context.AddMessage(llm.AssistantMessage(resp.Text()))

	return resp, nil
}

// RunWithMessages executes the agent with custom messages appended to the context history.
// The provided messages are added to the existing context history before making the call.
// Optional RunOption arguments can be passed to customize this specific call.
func (r *AgentRunner) RunWithMessages(ctx context.Context, messages []llm.Message, runOpts ...RunOption) (llm.Response[string], error) {
	// Apply run options
	cfg := &runConfig{}
	for _, opt := range runOpts {
		opt(cfg)
	}

	// Build options
	opts := make([]llm.Option, 0)

	if r.providerName != "" {
		opts = append(opts, llm.WithProvider(r.providerName))
	}
	if r.model != "" {
		opts = append(opts, llm.WithModel(r.model))
	}
	if r.temperature != nil {
		opts = append(opts, llm.WithTemperature(*r.temperature))
	}
	if r.maxTokens != nil {
		opts = append(opts, llm.WithMaxTokens(*r.maxTokens))
	}

	// Add agent's system message
	opts = append(opts, llm.WithSystemMessage(r.agent.ToSystemMessage()))

	// Add extra system message from run options (if any)
	if cfg.extraSystemMessage != "" {
		opts = append(opts, llm.WithSystemMessage(cfg.extraSystemMessage))
	}

	// Add filtered tools
	if len(r.filteredTools) > 0 {
		opts = append(opts, llm.WithTools(r.filteredTools...))
	}

	// Add runner-level extra LLM options
	opts = append(opts, r.extraLLMOpts...)

	// Add run-level extra LLM options
	opts = append(opts, cfg.extraLLMOpts...)

	// Build full message list: existing history + provided messages
	history := r.context.History()
	fullMessages := make([]llm.Message, 0, len(history)+len(messages))
	fullMessages = append(fullMessages, history...)
	fullMessages = append(fullMessages, messages...)

	// Make the LLM call
	resp, err := llm.CallMessages(ctx, fullMessages, opts...)
	if err != nil {
		return resp, err
	}

	// Add provided messages and response to context history
	r.context.AddMessages(messages...)
	r.context.AddMessage(llm.AssistantMessage(resp.Text()))

	return resp, nil
}

// Agent returns the underlying agent.
func (r *AgentRunner) Agent() *Agent {
	return r.agent
}

// FilteredTools returns the tools available to this agent runner.
func (r *AgentRunner) FilteredTools() []llm.Tool {
	return r.filteredTools
}

// Context returns the runner's context for accessing conversation history and state.
func (r *AgentRunner) Context() *AgentContext {
	return r.context
}

// ClearContext resets the runner's context, clearing all conversation history and state.
func (r *AgentRunner) ClearContext() {
	r.context.Clear()
}

// ClearHistory resets only the conversation history, keeping state.
func (r *AgentRunner) ClearHistory() {
	r.context.ClearHistory()
}

// ToOption converts an Agent to an llm.Option.
// This adds the agent's system message to the LLM call.
func (a *Agent) ToOption() llm.Option {
	return llm.WithSystemMessage(a.ToSystemMessage())
}

// FilterTools filters a list of tools to only include those allowed by this agent.
// If the agent has no tool restrictions, all tools are returned.
func (a *Agent) FilterTools(tools []llm.Tool) []llm.Tool {
	if len(a.Tools) == 0 {
		return tools
	}

	// Create a set of allowed tool names
	allowed := make(map[string]bool)
	for _, name := range a.Tools {
		allowed[name] = true
	}

	// Filter tools
	var filtered []llm.Tool
	for _, tool := range tools {
		if allowed[tool.Name()] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}
