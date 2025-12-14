package llm

import "github.com/i2y/bucephalus/provider"

// Message is an alias for provider.Message for convenience.
type Message = provider.Message

// Role is an alias for provider.Role for convenience.
type Role = provider.Role

// Role constants.
const (
	RoleSystem    = provider.RoleSystem
	RoleUser      = provider.RoleUser
	RoleAssistant = provider.RoleAssistant
	RoleTool      = provider.RoleTool
)

// SystemMessage creates a system message.
func SystemMessage(content string) Message {
	return Message{
		Role:    RoleSystem,
		Content: content,
	}
}

// UserMessage creates a user message.
func UserMessage(content string) Message {
	return Message{
		Role:    RoleUser,
		Content: content,
	}
}

// AssistantMessage creates an assistant message.
func AssistantMessage(content string) Message {
	return Message{
		Role:    RoleAssistant,
		Content: content,
	}
}

// AssistantMessageWithToolCalls creates an assistant message with tool calls.
func AssistantMessageWithToolCalls(content string, toolCalls []ToolCall) Message {
	providerToolCalls := make([]provider.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		providerToolCalls[i] = provider.ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
		}
	}
	return Message{
		Role:      RoleAssistant,
		Content:   content,
		ToolCalls: providerToolCalls,
	}
}

// ToolMessage creates a tool result message.
func ToolMessage(toolCallID, content string) Message {
	return Message{
		Role:    RoleTool,
		Content: content,
		ToolID:  toolCallID,
	}
}
