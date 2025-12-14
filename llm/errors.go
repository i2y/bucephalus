package llm

import (
	"errors"
	"fmt"
)

// Common errors.
var (
	// ErrProviderRequired is returned when WithProvider is not specified.
	ErrProviderRequired = errors.New("provider is required: use WithProvider option")

	// ErrModelRequired is returned when WithModel is not specified.
	ErrModelRequired = errors.New("model is required: use WithModel option")

	// ErrNotParsed is returned when Parsed() is called but no parsing occurred.
	ErrNotParsed = errors.New("response was not parsed: use CallParse to get structured output")
)

// ProviderError represents an error from the LLM provider.
type ProviderError struct {
	Provider   string
	StatusCode int
	Message    string
	Cause      error
}

func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s error (status %d): %s: %v",
			e.Provider, e.StatusCode, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s error (status %d): %s",
		e.Provider, e.StatusCode, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// ParseError represents a failure to parse the LLM response.
type ParseError struct {
	Content string
	Target  string
	Cause   error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse response as %s: %v", e.Target, e.Cause)
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}

// ToolError represents an error during tool execution.
type ToolError struct {
	ToolName string
	Cause    error
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("tool %q execution failed: %v", e.ToolName, e.Cause)
}

func (e *ToolError) Unwrap() error {
	return e.Cause
}

// ToolNotFoundError is returned when a tool is not found.
type ToolNotFoundError struct {
	Name string
}

func (e *ToolNotFoundError) Error() string {
	return fmt.Sprintf("tool not found: %q", e.Name)
}
