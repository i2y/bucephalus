// Package mcp provides integration with the Model Context Protocol (MCP).
// It allows Bucephalus to use tools from MCP servers.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/i2y/bucephalus/llm"
)

// Client wraps an MCP client for use with Bucephalus.
type Client struct {
	mcpClient *mcp.Client
	session   *mcp.ClientSession
	timeout   time.Duration
}

// Option configures the MCP client.
type Option func(*clientConfig)

type clientConfig struct {
	timeout time.Duration
}

// WithTimeout sets the timeout for tool execution.
func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// NewStdioClient creates an MCP client that communicates via stdio with a subprocess.
//
// Example:
//
//	client, err := mcp.NewStdioClient(ctx, "./my-mcp-server", nil)
//	if err != nil {
//	    return err
//	}
//	defer client.Close()
//
//	tools, err := client.Tools(ctx)
func NewStdioClient(ctx context.Context, command string, args []string, opts ...Option) (*Client, error) {
	cfg := &clientConfig{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Create the MCP client
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "bucephalus",
		Version: "0.1.0",
	}, nil)

	// Create command transport
	cmd := exec.Command(command, args...)
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	// Connect to the server
	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to MCP server: %w", err)
	}

	return &Client{
		mcpClient: mcpClient,
		session:   session,
		timeout:   cfg.timeout,
	}, nil
}

// Tools returns all tools from the MCP server as Bucephalus Tools.
//
// Example:
//
//	tools, err := client.Tools(ctx)
//	if err != nil {
//	    return err
//	}
//
//	resp, err := llm.Call(ctx, "Use the tools to help",
//	    llm.WithProvider("openai"),
//	    llm.WithModel("o4-mini"),
//	    llm.WithTools(tools...),
//	)
func (c *Client) Tools(ctx context.Context) ([]llm.Tool, error) {
	result, err := c.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("listing MCP tools: %w", err)
	}

	tools := make([]llm.Tool, 0, len(result.Tools))
	for i := range result.Tools {
		tools = append(tools, &mcpToolWrapper{
			client:  c,
			mcpTool: result.Tools[i],
		})
	}

	return tools, nil
}

// Close closes the MCP client connection.
func (c *Client) Close() error {
	return c.session.Close()
}

// mcpToolWrapper wraps an MCP tool to implement llm.Tool.
type mcpToolWrapper struct {
	client  *Client
	mcpTool *mcp.Tool
}

func (t *mcpToolWrapper) Name() string {
	return t.mcpTool.Name
}

func (t *mcpToolWrapper) Description() string {
	return t.mcpTool.Description
}

func (t *mcpToolWrapper) Parameters() *jsonschema.Schema {
	// Convert MCP input schema to jsonschema.Schema
	schemaBytes, err := json.Marshal(t.mcpTool.InputSchema)
	if err != nil {
		return &jsonschema.Schema{Type: "object"}
	}

	var schema jsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return &jsonschema.Schema{Type: "object"}
	}

	return &schema
}

func (t *mcpToolWrapper) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, t.client.timeout)
	defer cancel()

	// Parse arguments
	var arguments map[string]any
	if err := json.Unmarshal(args, &arguments); err != nil {
		return nil, fmt.Errorf("parsing arguments: %w", err)
	}

	// Call the MCP tool
	result, err := t.client.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      t.mcpTool.Name,
		Arguments: arguments,
	})
	if err != nil {
		return nil, fmt.Errorf("calling MCP tool: %w", err)
	}

	combined := processToolResult(result.Content)

	if result.IsError {
		return nil, fmt.Errorf("MCP tool error: %s", combined)
	}

	return combined, nil
}

// processToolResult extracts text content from MCP tool result.
// Multiple content items are joined with newlines.
// Non-text content (images, resources) are represented as descriptive text.
func processToolResult(content []mcp.Content) string {
	var parts []string
	for _, c := range content {
		switch item := c.(type) {
		case *mcp.TextContent:
			parts = append(parts, item.Text)
		case *mcp.ImageContent:
			// Return image info as text description
			parts = append(parts, fmt.Sprintf("[Image: %s, %d bytes]", item.MIMEType, len(item.Data)))
		case *mcp.EmbeddedResource:
			// Return resource info with URI
			if item.Resource != nil {
				parts = append(parts, fmt.Sprintf("[Resource: %s]", item.Resource.URI))
			} else {
				parts = append(parts, "[Resource: embedded]")
			}
		}
	}
	return strings.Join(parts, "\n")
}

// ToolsFromMCP is a convenience function to get tools from an MCP server.
//
// Example:
//
//	tools, cleanup, err := mcp.ToolsFromMCP(ctx, "./my-mcp-server", nil)
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
//
//	resp, err := llm.Call(ctx, "Help me", llm.WithTools(tools...))
func ToolsFromMCP(ctx context.Context, command string, args []string, opts ...Option) ([]llm.Tool, func() error, error) {
	mcpClient, err := NewStdioClient(ctx, command, args, opts...)
	if err != nil {
		return nil, nil, err
	}

	tools, err := mcpClient.Tools(ctx)
	if err != nil {
		_ = mcpClient.Close()
		return nil, nil, err
	}

	return tools, mcpClient.Close, nil
}
