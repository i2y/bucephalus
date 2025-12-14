package llm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderError(t *testing.T) {
	tests := []struct {
		name       string
		err        *ProviderError
		wantSubstr []string
	}{
		{
			name: "error without cause",
			err: &ProviderError{
				Provider:   "openai",
				StatusCode: 400,
				Message:    "Bad request",
			},
			wantSubstr: []string{"openai", "400", "Bad request"},
		},
		{
			name: "error with cause",
			err: &ProviderError{
				Provider:   "anthropic",
				StatusCode: 500,
				Message:    "Internal error",
				Cause:      errors.New("underlying error"),
			},
			wantSubstr: []string{"anthropic", "500", "Internal error", "underlying error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.wantSubstr {
				assert.Contains(t, errStr, substr)
			}
		})
	}
}

func TestProviderError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := &ProviderError{
		Provider:   "test",
		StatusCode: 500,
		Message:    "error",
		Cause:      cause,
	}

	assert.ErrorIs(t, err, cause)
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestProviderError_Unwrap_NilCause(t *testing.T) {
	err := &ProviderError{
		Provider:   "test",
		StatusCode: 500,
		Message:    "error",
		Cause:      nil,
	}

	assert.Nil(t, errors.Unwrap(err))
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name       string
		err        *ParseError
		wantSubstr []string
	}{
		{
			name: "parse error with JSON",
			err: &ParseError{
				Content: `{"invalid": json}`,
				Target:  "MyStruct",
				Cause:   errors.New("unexpected character"),
			},
			wantSubstr: []string{"MyStruct", "unexpected character"},
		},
		{
			name: "parse error with type name",
			err: &ParseError{
				Content: "some content",
				Target:  "Recipe",
				Cause:   errors.New("missing field"),
			},
			wantSubstr: []string{"Recipe", "missing field"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.wantSubstr {
				assert.Contains(t, errStr, substr)
			}
		})
	}
}

func TestParseError_Unwrap(t *testing.T) {
	cause := errors.New("json parse error")
	err := &ParseError{
		Content: "content",
		Target:  "Target",
		Cause:   cause,
	}

	assert.ErrorIs(t, err, cause)
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestToolError(t *testing.T) {
	tests := []struct {
		name       string
		err        *ToolError
		wantSubstr []string
	}{
		{
			name: "tool error",
			err: &ToolError{
				ToolName: "get_weather",
				Cause:    errors.New("API timeout"),
			},
			wantSubstr: []string{"get_weather", "API timeout"},
		},
		{
			name: "tool error with quoted name",
			err: &ToolError{
				ToolName: "calculate",
				Cause:    errors.New("division by zero"),
			},
			wantSubstr: []string{"calculate", "division by zero"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.wantSubstr {
				assert.Contains(t, errStr, substr)
			}
		})
	}
}

func TestToolError_Unwrap(t *testing.T) {
	cause := errors.New("execution failed")
	err := &ToolError{
		ToolName: "test_tool",
		Cause:    cause,
	}

	assert.ErrorIs(t, err, cause)
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestToolNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
	}{
		{
			name:     "simple tool name",
			toolName: "get_weather",
		},
		{
			name:     "tool with special chars",
			toolName: "my-tool_v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ToolNotFoundError{Name: tt.toolName}
			errStr := err.Error()

			assert.Contains(t, errStr, tt.toolName)
			assert.Contains(t, errStr, "not found")
		})
	}
}

func TestCommonErrors(t *testing.T) {
	// Test that common errors are properly defined
	assert.NotNil(t, ErrProviderRequired)
	assert.NotNil(t, ErrModelRequired)
	assert.NotNil(t, ErrNotParsed)

	// Test error messages contain useful info
	assert.Contains(t, ErrProviderRequired.Error(), "provider")
	assert.Contains(t, ErrModelRequired.Error(), "model")
	assert.Contains(t, ErrNotParsed.Error(), "parsed")
}

func TestErrorsAreCompatibleWithStdErrors(t *testing.T) {
	// Verify our custom errors work with errors.Is and errors.As
	cause := errors.New("root")

	t.Run("ProviderError", func(t *testing.T) {
		err := &ProviderError{Provider: "test", Cause: cause}
		var provErr *ProviderError
		assert.True(t, errors.As(err, &provErr))
		assert.ErrorIs(t, err, cause)
	})

	t.Run("ParseError", func(t *testing.T) {
		err := &ParseError{Target: "test", Cause: cause}
		var parseErr *ParseError
		assert.True(t, errors.As(err, &parseErr))
		assert.ErrorIs(t, err, cause)
	})

	t.Run("ToolError", func(t *testing.T) {
		err := &ToolError{ToolName: "test", Cause: cause}
		var toolErr *ToolError
		assert.True(t, errors.As(err, &toolErr))
		assert.ErrorIs(t, err, cause)
	})

	t.Run("ToolNotFoundError", func(t *testing.T) {
		err := &ToolNotFoundError{Name: "test"}
		var notFoundErr *ToolNotFoundError
		assert.True(t, errors.As(err, &notFoundErr))
	})
}
