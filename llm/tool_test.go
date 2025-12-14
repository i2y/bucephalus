package llm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test input/output types
type TestInput struct {
	Name  string `json:"name" jsonschema:"required,description=The name"`
	Count int    `json:"count,omitempty"`
}

type TestOutput struct {
	Result string `json:"result"`
	Value  int    `json:"value"`
}

func TestNewTool(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		description string
	}{
		{
			name:        "simple tool",
			toolName:    "test_tool",
			description: "A test tool",
		},
		{
			name:        "tool with long description",
			toolName:    "another_tool",
			description: "This is a much longer description that explains what the tool does in detail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := NewTool(tt.toolName, tt.description,
				func(ctx context.Context, in TestInput) (TestOutput, error) {
					return TestOutput{Result: in.Name, Value: in.Count}, nil
				})

			require.NoError(t, err)
			assert.Equal(t, tt.toolName, tool.Name())
			assert.Equal(t, tt.description, tool.Description())
			assert.NotNil(t, tool.Parameters())
		})
	}
}

func TestTypedTool_Execute(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr bool
		check   func(t *testing.T, result any)
	}{
		{
			name:    "valid JSON args",
			args:    `{"name": "test", "count": 42}`,
			wantErr: false,
			check: func(t *testing.T, result any) {
				out, ok := result.(TestOutput)
				require.True(t, ok)
				assert.Equal(t, "test", out.Result)
				assert.Equal(t, 42, out.Value)
			},
		},
		{
			name:    "minimal args",
			args:    `{"name": "minimal"}`,
			wantErr: false,
			check: func(t *testing.T, result any) {
				out, ok := result.(TestOutput)
				require.True(t, ok)
				assert.Equal(t, "minimal", out.Result)
				assert.Equal(t, 0, out.Value)
			},
		},
		{
			name:    "invalid JSON",
			args:    `not valid json`,
			wantErr: true,
		},
		{
			name:    "empty JSON",
			args:    `{}`,
			wantErr: false,
			check: func(t *testing.T, result any) {
				out, ok := result.(TestOutput)
				require.True(t, ok)
				assert.Equal(t, "", out.Result)
			},
		},
	}

	tool, err := NewTool("test", "test tool",
		func(ctx context.Context, in TestInput) (TestOutput, error) {
			return TestOutput{Result: in.Name, Value: in.Count}, nil
		})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := tool.Execute(ctx, json.RawMessage(tt.args))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestTypedTool_Execute_FunctionError(t *testing.T) {
	expectedErr := errors.New("function error")

	tool, err := NewTool("error_tool", "tool that errors",
		func(ctx context.Context, in TestInput) (TestOutput, error) {
			return TestOutput{}, expectedErr
		})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = tool.Execute(ctx, json.RawMessage(`{"name": "test"}`))
	assert.ErrorIs(t, err, expectedErr)
}

func TestTypedTool_TypedCall(t *testing.T) {
	tool, err := NewTool("typed_call_test", "test typed call",
		func(ctx context.Context, in TestInput) (TestOutput, error) {
			return TestOutput{Result: in.Name + "_processed", Value: in.Count * 2}, nil
		})
	require.NoError(t, err)

	ctx := context.Background()
	result, err := tool.TypedCall(ctx, TestInput{Name: "direct", Count: 5})

	require.NoError(t, err)
	assert.Equal(t, "direct_processed", result.Result)
	assert.Equal(t, 10, result.Value)
}

func TestTypedTool_Parameters_HasCorrectSchema(t *testing.T) {
	tool, err := NewTool("schema_test", "test schema",
		func(ctx context.Context, in TestInput) (TestOutput, error) {
			return TestOutput{}, nil
		})
	require.NoError(t, err)

	params := tool.Parameters()
	require.NotNil(t, params)

	// Check properties exist
	assert.NotNil(t, params.Properties)
	// Check that the properties contain expected keys
	_, hasName := params.Properties.Get("name")
	_, hasCount := params.Properties.Get("count")
	assert.True(t, hasName, "schema should have 'name' property")
	assert.True(t, hasCount, "schema should have 'count' property")
}

func TestMustNewTool(t *testing.T) {
	t.Run("does not panic on valid tool", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tool := MustNewTool("must_test", "test",
				func(ctx context.Context, in TestInput) (TestOutput, error) {
					return TestOutput{}, nil
				})
			assert.NotNil(t, tool)
		})
	})
}

func TestToolRegistry(t *testing.T) {
	t.Run("register and get single tool", func(t *testing.T) {
		registry := NewToolRegistry()
		tool := MustNewTool("tool1", "first tool",
			func(ctx context.Context, in TestInput) (TestOutput, error) {
				return TestOutput{}, nil
			})

		registry.Register(tool)

		got, ok := registry.Get("tool1")
		assert.True(t, ok)
		assert.Equal(t, "tool1", got.Name())
	})

	t.Run("register multiple tools", func(t *testing.T) {
		registry := NewToolRegistry()
		tool1 := MustNewTool("tool1", "first", func(ctx context.Context, in TestInput) (TestOutput, error) { return TestOutput{}, nil })
		tool2 := MustNewTool("tool2", "second", func(ctx context.Context, in TestInput) (TestOutput, error) { return TestOutput{}, nil })
		tool3 := MustNewTool("tool3", "third", func(ctx context.Context, in TestInput) (TestOutput, error) { return TestOutput{}, nil })

		registry.Register(tool1, tool2, tool3)

		all := registry.All()
		assert.Len(t, all, 3)
	})

	t.Run("get non-existent tool", func(t *testing.T) {
		registry := NewToolRegistry()

		_, ok := registry.Get("nonexistent")
		assert.False(t, ok)
	})

	t.Run("overwrite existing tool", func(t *testing.T) {
		registry := NewToolRegistry()
		tool1 := MustNewTool("tool", "first", func(ctx context.Context, in TestInput) (TestOutput, error) { return TestOutput{Result: "first"}, nil })
		tool2 := MustNewTool("tool", "second", func(ctx context.Context, in TestInput) (TestOutput, error) { return TestOutput{Result: "second"}, nil })

		registry.Register(tool1)
		registry.Register(tool2)

		got, ok := registry.Get("tool")
		require.True(t, ok)
		assert.Equal(t, "second", got.Description())
	})
}

func TestExecuteToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		toolCalls []ToolCall
		setup     func(*ToolRegistry)
		wantErr   bool
		checkMsgs func(t *testing.T, msgs []Message)
	}{
		{
			name:      "empty tool calls",
			toolCalls: []ToolCall{},
			setup:     func(r *ToolRegistry) {},
			wantErr:   false,
			checkMsgs: func(t *testing.T, msgs []Message) {
				assert.Nil(t, msgs)
			},
		},
		{
			name: "single successful tool call",
			toolCalls: []ToolCall{
				{ID: "call1", Name: "echo", Arguments: `{"name": "hello"}`},
			},
			setup: func(r *ToolRegistry) {
				r.Register(MustNewTool("echo", "echoes input",
					func(ctx context.Context, in TestInput) (string, error) {
						return "echoed: " + in.Name, nil
					}))
			},
			wantErr: false,
			checkMsgs: func(t *testing.T, msgs []Message) {
				require.Len(t, msgs, 1)
				assert.Equal(t, RoleTool, msgs[0].Role)
				assert.Equal(t, "call1", msgs[0].ToolID)
				assert.Equal(t, "echoed: hello", msgs[0].Content)
			},
		},
		{
			name: "tool returns struct",
			toolCalls: []ToolCall{
				{ID: "call1", Name: "struct_tool", Arguments: `{"name": "test", "count": 5}`},
			},
			setup: func(r *ToolRegistry) {
				r.Register(MustNewTool("struct_tool", "returns struct",
					func(ctx context.Context, in TestInput) (TestOutput, error) {
						return TestOutput{Result: in.Name, Value: in.Count}, nil
					}))
			},
			wantErr: false,
			checkMsgs: func(t *testing.T, msgs []Message) {
				require.Len(t, msgs, 1)
				// Should be JSON marshaled
				var out TestOutput
				err := json.Unmarshal([]byte(msgs[0].Content), &out)
				require.NoError(t, err)
				assert.Equal(t, "test", out.Result)
				assert.Equal(t, 5, out.Value)
			},
		},
		{
			name: "tool not found",
			toolCalls: []ToolCall{
				{ID: "call1", Name: "nonexistent", Arguments: `{}`},
			},
			setup:   func(r *ToolRegistry) {},
			wantErr: true,
		},
		{
			name: "multiple tool calls",
			toolCalls: []ToolCall{
				{ID: "call1", Name: "tool1", Arguments: `{"name": "first"}`},
				{ID: "call2", Name: "tool2", Arguments: `{"name": "second"}`},
			},
			setup: func(r *ToolRegistry) {
				r.Register(MustNewTool("tool1", "first tool",
					func(ctx context.Context, in TestInput) (string, error) {
						return "result1", nil
					}))
				r.Register(MustNewTool("tool2", "second tool",
					func(ctx context.Context, in TestInput) (string, error) {
						return "result2", nil
					}))
			},
			wantErr: false,
			checkMsgs: func(t *testing.T, msgs []Message) {
				require.Len(t, msgs, 2)
				assert.Equal(t, "call1", msgs[0].ToolID)
				assert.Equal(t, "call2", msgs[1].ToolID)
			},
		},
		{
			name: "tool execution error included in message",
			toolCalls: []ToolCall{
				{ID: "call1", Name: "error_tool", Arguments: `{"name": "test"}`},
			},
			setup: func(r *ToolRegistry) {
				r.Register(MustNewTool("error_tool", "tool that errors",
					func(ctx context.Context, in TestInput) (string, error) {
						return "", errors.New("tool execution failed")
					}))
			},
			wantErr: false,
			checkMsgs: func(t *testing.T, msgs []Message) {
				require.Len(t, msgs, 1)
				assert.Contains(t, msgs[0].Content, "Error:")
				assert.Contains(t, msgs[0].Content, "tool execution failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewToolRegistry()
			tt.setup(registry)

			ctx := context.Background()
			msgs, err := ExecuteToolCalls(ctx, tt.toolCalls, registry)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkMsgs != nil {
				tt.checkMsgs(t, msgs)
			}
		})
	}
}
