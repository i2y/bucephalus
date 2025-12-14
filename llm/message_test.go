package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemMessage(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "simple system message",
			content: "You are a helpful assistant.",
		},
		{
			name:    "empty content",
			content: "",
		},
		{
			name:    "multiline content",
			content: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := SystemMessage(tt.content)

			assert.Equal(t, RoleSystem, msg.Role)
			assert.Equal(t, tt.content, msg.Content)
			assert.Empty(t, msg.ToolCalls)
			assert.Empty(t, msg.ToolID)
		})
	}
}

func TestUserMessage(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "simple user message",
			content: "Hello, how are you?",
		},
		{
			name:    "empty content",
			content: "",
		},
		{
			name:    "message with special characters",
			content: "Special chars: @#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := UserMessage(tt.content)

			assert.Equal(t, RoleUser, msg.Role)
			assert.Equal(t, tt.content, msg.Content)
			assert.Empty(t, msg.ToolCalls)
			assert.Empty(t, msg.ToolID)
		})
	}
}

func TestAssistantMessage(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "simple assistant message",
			content: "I'm doing well, thank you!",
		},
		{
			name:    "empty content",
			content: "",
		},
		{
			name:    "long response",
			content: "This is a very long response that contains a lot of text and goes on for quite a while to test how the system handles longer content.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := AssistantMessage(tt.content)

			assert.Equal(t, RoleAssistant, msg.Role)
			assert.Equal(t, tt.content, msg.Content)
			assert.Empty(t, msg.ToolCalls)
			assert.Empty(t, msg.ToolID)
		})
	}
}

func TestAssistantMessageWithToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		toolCalls []ToolCall
	}{
		{
			name:    "single tool call",
			content: "Let me check the weather.",
			toolCalls: []ToolCall{
				{ID: "call_1", Name: "get_weather", Arguments: `{"city": "Tokyo"}`},
			},
		},
		{
			name:    "multiple tool calls",
			content: "",
			toolCalls: []ToolCall{
				{ID: "call_1", Name: "tool_a", Arguments: `{"arg": "a"}`},
				{ID: "call_2", Name: "tool_b", Arguments: `{"arg": "b"}`},
				{ID: "call_3", Name: "tool_c", Arguments: `{"arg": "c"}`},
			},
		},
		{
			name:      "empty tool calls",
			content:   "No tools needed.",
			toolCalls: []ToolCall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := AssistantMessageWithToolCalls(tt.content, tt.toolCalls)

			assert.Equal(t, RoleAssistant, msg.Role)
			assert.Equal(t, tt.content, msg.Content)
			assert.Len(t, msg.ToolCalls, len(tt.toolCalls))

			for i, tc := range tt.toolCalls {
				assert.Equal(t, tc.ID, msg.ToolCalls[i].ID)
				assert.Equal(t, tc.Name, msg.ToolCalls[i].Name)
				assert.Equal(t, tc.Arguments, msg.ToolCalls[i].Arguments)
			}
		})
	}
}

func TestToolMessage(t *testing.T) {
	tests := []struct {
		name       string
		toolCallID string
		content    string
	}{
		{
			name:       "simple tool result",
			toolCallID: "call_123",
			content:    `{"temperature": 72, "conditions": "sunny"}`,
		},
		{
			name:       "string content",
			toolCallID: "call_456",
			content:    "The weather is nice today.",
		},
		{
			name:       "error content",
			toolCallID: "call_789",
			content:    "Error: Unable to fetch data",
		},
		{
			name:       "empty content",
			toolCallID: "call_empty",
			content:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ToolMessage(tt.toolCallID, tt.content)

			assert.Equal(t, RoleTool, msg.Role)
			assert.Equal(t, tt.content, msg.Content)
			assert.Equal(t, tt.toolCallID, msg.ToolID)
			assert.Empty(t, msg.ToolCalls)
		})
	}
}

func TestRoleConstants(t *testing.T) {
	// Verify role constants have expected values
	tests := []struct {
		name     string
		role     Role
		expected string
	}{
		{"system role", RoleSystem, "system"},
		{"user role", RoleUser, "user"},
		{"assistant role", RoleAssistant, "assistant"},
		{"tool role", RoleTool, "tool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, Role(tt.expected), tt.role)
		})
	}
}
