package plugin

import (
	"sync"

	"github.com/i2y/bucephalus/llm"
)

// AgentContext maintains conversation history and state for an agent.
// It provides thread-safe access to conversation history and arbitrary state storage.
// Contexts can have parent contexts for inheritance (e.g., sub-agents inheriting from parent).
type AgentContext struct {
	history []llm.Message  // Conversation history
	state   map[string]any // Arbitrary state storage
	parent  *AgentContext  // Parent context (for inheritance)
	mu      sync.RWMutex   // Thread safety
}

// NewAgentContext creates a new empty context.
func NewAgentContext() *AgentContext {
	return &AgentContext{
		history: make([]llm.Message, 0),
		state:   make(map[string]any),
	}
}

// NewChildContext creates a child context that inherits state from this context.
// The child has its own history but can access parent's state through GetState.
func (c *AgentContext) NewChildContext() *AgentContext {
	return &AgentContext{
		history: make([]llm.Message, 0),
		state:   make(map[string]any),
		parent:  c,
	}
}

// History returns a copy of the conversation history.
func (c *AgentContext) History() []llm.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]llm.Message, len(c.history))
	copy(result, c.history)
	return result
}

// HistoryLen returns the number of messages in the history.
func (c *AgentContext) HistoryLen() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.history)
}

// AddMessage adds a message to the conversation history.
func (c *AgentContext) AddMessage(msg llm.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.history = append(c.history, msg)
}

// AddMessages adds multiple messages to the conversation history.
func (c *AgentContext) AddMessages(msgs ...llm.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.history = append(c.history, msgs...)
}

// SetState stores a value in the context with the given key.
func (c *AgentContext) SetState(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state[key] = value
}

// GetState retrieves a value from the context.
// If the key is not found in this context, it checks the parent context recursively.
// Returns the value and true if found, or nil and false if not found.
func (c *AgentContext) GetState(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// First check this context
	if value, ok := c.state[key]; ok {
		return value, true
	}

	// Then check parent context if exists
	if c.parent != nil {
		return c.parent.GetState(key)
	}

	return nil, false
}

// DeleteState removes a value from the context.
// Note: This only removes from this context, not from parent contexts.
func (c *AgentContext) DeleteState(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.state, key)
}

// HasState checks if a key exists in this context or its parents.
func (c *AgentContext) HasState(key string) bool {
	_, ok := c.GetState(key)
	return ok
}

// StateKeys returns all keys in this context (not including parent keys).
func (c *AgentContext) StateKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.state))
	for k := range c.state {
		keys = append(keys, k)
	}
	return keys
}

// Clear resets both conversation history and state.
// Note: This does not affect the parent context.
func (c *AgentContext) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.history = make([]llm.Message, 0)
	c.state = make(map[string]any)
}

// ClearHistory resets only the conversation history, keeping state.
func (c *AgentContext) ClearHistory() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.history = make([]llm.Message, 0)
}

// ClearState resets only the state, keeping conversation history.
func (c *AgentContext) ClearState() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = make(map[string]any)
}

// Parent returns the parent context, or nil if this is a root context.
func (c *AgentContext) Parent() *AgentContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.parent
}

// SetParent sets the parent context.
func (c *AgentContext) SetParent(parent *AgentContext) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.parent = parent
}

// Clone creates a deep copy of this context including history and state.
// The clone does not share the same parent reference.
func (c *AgentContext) Clone() *AgentContext {
	c.mu.RLock()
	defer c.mu.RUnlock()

	clone := &AgentContext{
		history: make([]llm.Message, len(c.history)),
		state:   make(map[string]any, len(c.state)),
		parent:  c.parent, // Share parent reference
	}

	copy(clone.history, c.history)
	for k, v := range c.state {
		clone.state[k] = v
	}

	return clone
}

// LastMessage returns the last message in the history, or nil if empty.
func (c *AgentContext) LastMessage() *llm.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.history) == 0 {
		return nil
	}
	msg := c.history[len(c.history)-1]
	return &msg
}

// LastNMessages returns the last n messages from history.
// If n is greater than history length, returns all messages.
func (c *AgentContext) LastNMessages(n int) []llm.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if n >= len(c.history) {
		result := make([]llm.Message, len(c.history))
		copy(result, c.history)
		return result
	}

	start := len(c.history) - n
	result := make([]llm.Message, n)
	copy(result, c.history[start:])
	return result
}
