package mcp

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

func TestProcessToolResult(t *testing.T) {
	tests := []struct {
		name     string
		content  []mcp.Content
		expected string
	}{
		{
			name:     "empty content",
			content:  []mcp.Content{},
			expected: "",
		},
		{
			name: "single text content",
			content: []mcp.Content{
				&mcp.TextContent{Text: "Hello, World!"},
			},
			expected: "Hello, World!",
		},
		{
			name: "multiple text contents joined with newline",
			content: []mcp.Content{
				&mcp.TextContent{Text: "Line 1"},
				&mcp.TextContent{Text: "Line 2"},
				&mcp.TextContent{Text: "Line 3"},
			},
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name: "image content",
			content: []mcp.Content{
				&mcp.ImageContent{
					MIMEType: "image/png",
					Data:     []byte("base64encodeddata"), // 17 bytes
				},
			},
			expected: "[Image: image/png, 17 bytes]",
		},
		{
			name: "embedded resource",
			content: []mcp.Content{
				&mcp.EmbeddedResource{
					Resource: &mcp.ResourceContents{
						URI: "file:///path/to/resource.txt",
					},
				},
			},
			expected: "[Resource: file:///path/to/resource.txt]",
		},
		{
			name: "embedded resource with nil resource",
			content: []mcp.Content{
				&mcp.EmbeddedResource{
					Resource: nil,
				},
			},
			expected: "[Resource: embedded]",
		},
		{
			name: "mixed content types",
			content: []mcp.Content{
				&mcp.TextContent{Text: "Here is the data:"},
				&mcp.ImageContent{
					MIMEType: "image/jpeg",
					Data:     []byte("jpeg_data_here"),
				},
				&mcp.TextContent{Text: "And a resource:"},
				&mcp.EmbeddedResource{
					Resource: &mcp.ResourceContents{
						URI: "file:///data.json",
					},
				},
			},
			expected: "Here is the data:\n[Image: image/jpeg, 14 bytes]\nAnd a resource:\n[Resource: file:///data.json]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processToolResult(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}
