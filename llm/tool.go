package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"

	"github.com/i2y/bucephalus/schema"
)

// Tool represents an executable tool that the LLM can call.
// This interface allows for heterogeneous collections of tools.
type Tool interface {
	// Name returns the tool's name as seen by the LLM.
	Name() string

	// Description returns the tool's description for the LLM.
	Description() string

	// Parameters returns the JSON schema for the tool's parameters.
	Parameters() *jsonschema.Schema

	// Execute runs the tool with the given JSON arguments.
	Execute(ctx context.Context, args json.RawMessage) (any, error)
}

// TypedTool provides type-safe tool creation with auto-generated schema.
// In is the input type, Out is the output type.
type TypedTool[In any, Out any] struct {
	name        string
	description string
	fn          func(ctx context.Context, in In) (Out, error)
	schema      *jsonschema.Schema
}

// NewTool creates a type-safe tool from a function.
// The input type In is used to generate the JSON schema automatically.
//
// Example:
//
//	type WeatherInput struct {
//	    City string `json:"city" jsonschema:"required,description=City name"`
//	}
//
//	type WeatherOutput struct {
//	    Temperature float64 `json:"temperature"`
//	    Conditions  string  `json:"conditions"`
//	}
//
//	weatherTool, err := llm.NewTool("get_weather", "Get weather for a city",
//	    func(ctx context.Context, in WeatherInput) (WeatherOutput, error) {
//	        return WeatherOutput{Temperature: 72.5, Conditions: "Sunny"}, nil
//	    },
//	)
func NewTool[In any, Out any](
	name, description string,
	fn func(ctx context.Context, in In) (Out, error),
) (*TypedTool[In, Out], error) {
	var zero In
	paramSchema := schema.Reflector.Reflect(&zero)

	return &TypedTool[In, Out]{
		name:        name,
		description: description,
		fn:          fn,
		schema:      paramSchema,
	}, nil
}

// MustNewTool is like NewTool but panics on error.
// Useful for package-level tool definitions.
func MustNewTool[In any, Out any](
	name, description string,
	fn func(ctx context.Context, in In) (Out, error),
) *TypedTool[In, Out] {
	t, err := NewTool(name, description, fn)
	if err != nil {
		panic(err)
	}
	return t
}

// Name returns the tool's name.
func (t *TypedTool[In, Out]) Name() string {
	return t.name
}

// Description returns the tool's description.
func (t *TypedTool[In, Out]) Description() string {
	return t.description
}

// Parameters returns the JSON schema for the tool's parameters.
func (t *TypedTool[In, Out]) Parameters() *jsonschema.Schema {
	return t.schema
}

// Execute runs the tool with the given JSON arguments.
// Implements the Tool interface.
func (t *TypedTool[In, Out]) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	var input In
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool arguments: %w", err)
	}
	return t.fn(ctx, input)
}

// TypedCall provides a type-safe way to call the tool directly.
// This bypasses JSON marshaling when you have the typed input.
func (t *TypedTool[In, Out]) TypedCall(ctx context.Context, input In) (Out, error) {
	return t.fn(ctx, input)
}

// ToolRegistry manages a collection of tools.
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(tools ...Tool) {
	for _, t := range tools {
		r.tools[t.Name()] = t
	}
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools.
func (r *ToolRegistry) All() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// ExecuteToolCalls executes tool calls and returns tool result messages.
func ExecuteToolCalls(ctx context.Context, toolCalls []ToolCall, registry *ToolRegistry) ([]Message, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	messages := make([]Message, 0, len(toolCalls))

	for _, tc := range toolCalls {
		tool, ok := registry.Get(tc.Name)
		if !ok {
			return nil, &ToolNotFoundError{Name: tc.Name}
		}

		result, err := tool.Execute(ctx, json.RawMessage(tc.Arguments))
		var content string
		if err != nil {
			content = fmt.Sprintf("Error: %v", err)
		} else {
			// Marshal result to JSON if it's not already a string
			if s, ok := result.(string); ok {
				content = s
			} else {
				bytes, err := json.Marshal(result)
				if err != nil {
					content = fmt.Sprintf("Error marshaling result: %v", err)
				} else {
					content = string(bytes)
				}
			}
		}

		messages = append(messages, ToolMessage(tc.ID, content))
	}

	return messages, nil
}
