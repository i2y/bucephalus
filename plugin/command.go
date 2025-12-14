package plugin

import (
	"errors"
	"strings"

	"github.com/i2y/bucephalus/llm"
)

// ExpandedCommand represents an expanded command ready for LLM call.
type ExpandedCommand struct {
	Command       *Command // The original command
	SystemMessage string   // Command content with $ARGUMENTS replaced
	UserMessage   string   // The arguments or original input
	Arguments     string   // Extracted arguments after command name
}

var (
	// ErrNotACommand is returned when input doesn't start with a slash command.
	ErrNotACommand = errors.New("input is not a slash command")
	// ErrCommandNotFound is returned when the command doesn't exist in the plugin.
	ErrCommandNotFound = errors.New("command not found")
)

// IsCommand checks if input starts with a slash command.
// Returns true for inputs like "/greet", "/hello world", etc.
func (p *Plugin) IsCommand(input string) bool {
	input = strings.TrimSpace(input)
	return strings.HasPrefix(input, "/")
}

// ExpandCommand expands a command from user input.
// Input: "/greet John" → finds "greet" command, extracts "John" as argument.
// The command's Content is used as SystemMessage with $ARGUMENTS replaced.
func (p *Plugin) ExpandCommand(input string) (*ExpandedCommand, error) {
	input = strings.TrimSpace(input)

	if !strings.HasPrefix(input, "/") {
		return nil, ErrNotACommand
	}

	// Remove leading slash
	input = strings.TrimPrefix(input, "/")

	// Split into command name and arguments
	parts := strings.SplitN(input, " ", 2)
	cmdName := parts[0]
	arguments := ""
	if len(parts) > 1 {
		arguments = strings.TrimSpace(parts[1])
	}

	// Find the command
	cmd := p.GetCommand(cmdName)
	if cmd == nil {
		return nil, ErrCommandNotFound
	}

	// Expand the command content with arguments
	systemMessage := cmd.Content
	if arguments != "" {
		systemMessage = strings.ReplaceAll(systemMessage, "$ARGUMENTS", arguments)
	}

	return &ExpandedCommand{
		Command:       cmd,
		SystemMessage: systemMessage,
		UserMessage:   arguments,
		Arguments:     arguments,
	}, nil
}

// ParseCommandInput parses a potential command input and returns the command name and arguments.
// Returns empty strings if the input is not a command.
func ParseCommandInput(input string) (cmdName, arguments string) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", ""
	}

	input = strings.TrimPrefix(input, "/")
	parts := strings.SplitN(input, " ", 2)
	cmdName = parts[0]
	if len(parts) > 1 {
		arguments = strings.TrimSpace(parts[1])
	}
	return cmdName, arguments
}

// ToOption converts an ExpandedCommand to an llm.Option.
// This adds the expanded command's system message to the LLM call.
func (e *ExpandedCommand) ToOption() llm.Option {
	return llm.WithSystemMessage(e.SystemMessage)
}

// ToOption converts a Command to an llm.Option.
// This adds the command's content as system message to the LLM call.
func (c *Command) ToOption() llm.Option {
	return llm.WithSystemMessage(c.ToSystemMessage())
}

// ToOptionWithArgs converts a Command to an llm.Option with argument substitution.
// The $ARGUMENTS placeholder in the command content is replaced with the provided arguments.
func (c *Command) ToOptionWithArgs(arguments string) llm.Option {
	content := c.Content
	if arguments != "" {
		content = strings.ReplaceAll(content, "$ARGUMENTS", arguments)
	}
	return llm.WithSystemMessage(content)
}

// ProcessInput processes user input and returns the appropriate llm.Option.
// If the input is a slash command (e.g., "/greet John"), it expands the command.
// If the input is not a command, it returns nil and the original input.
//
// Usage:
//
//	opt, userMsg, err := plugin.ProcessInput("/greet John")
//	if err != nil { ... }
//	if opt != nil {
//	    resp, _ := llm.Call(ctx, userMsg, opt, otherOpts...)
//	} else {
//	    resp, _ := llm.Call(ctx, userMsg, otherOpts...)
//	}
func (p *Plugin) ProcessInput(input string) (opt llm.Option, userMessage string, err error) {
	if !p.IsCommand(input) {
		return nil, input, nil
	}

	expanded, err := p.ExpandCommand(input)
	if err != nil {
		return nil, input, err
	}

	// If there are arguments, use them as user message; otherwise use empty
	userMessage = expanded.Arguments
	if userMessage == "" {
		userMessage = input // fallback to original input
	}

	return expanded.ToOption(), userMessage, nil
}
