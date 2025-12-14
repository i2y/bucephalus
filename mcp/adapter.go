// Package mcp provides integration with the Model Context Protocol (MCP).
// It allows Bucephalus to use tools from MCP servers.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/i2y/bucephalus/llm"
)

// Client wraps an MCP client for use with Bucephalus.
type Client struct {
	mcpClient   *client.Client
	timeout     time.Duration
	initialized bool
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
//	client, err := mcp.NewStdioClient("./my-mcp-server")
//	if err != nil {
//	    return err
//	}
//	defer client.Close()
//
//	tools, err := client.Tools(ctx)
func NewStdioClient(command string, args []string, opts ...Option) (*Client, error) {
	cfg := &clientConfig{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	mcpClient, err := client.NewStdioMCPClient(command, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("creating MCP client: %w", err)
	}

	return &Client{
		mcpClient: mcpClient,
		timeout:   cfg.timeout,
	}, nil
}

// Initialize initializes the MCP connection.
// This is called automatically by Tools() if not already initialized.
func (c *Client) Initialize(ctx context.Context) error {
	if c.initialized {
		return nil
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "bucephalus",
		Version: "0.1.0",
	}

	_, err := c.mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		return fmt.Errorf("initializing MCP: %w", err)
	}

	c.initialized = true
	return nil
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
//	    llm.WithModel("gpt-4o"),
//	    llm.WithTools(tools...),
//	)
func (c *Client) Tools(ctx context.Context) ([]llm.Tool, error) {
	if err := c.Initialize(ctx); err != nil {
		return nil, err
	}

	listReq := mcp.ListToolsRequest{}
	result, err := c.mcpClient.ListTools(ctx, listReq)
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
	return c.mcpClient.Close()
}

// mcpToolWrapper wraps an MCP tool to implement llm.Tool.
type mcpToolWrapper struct {
	client  *Client
	mcpTool mcp.Tool
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
	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = t.mcpTool.Name
	callReq.Params.Arguments = arguments

	result, err := t.client.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return nil, fmt.Errorf("calling MCP tool: %w", err)
	}

	// Extract text content from result
	var content string
	for _, c := range result.Content {
		if textContent, ok := c.(mcp.TextContent); ok {
			content += textContent.Text
		}
	}

	if result.IsError {
		return nil, fmt.Errorf("MCP tool error: %s", content)
	}

	return content, nil
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
	mcpClient, err := NewStdioClient(command, args, opts...)
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
