// Package main demonstrates MCP integration with bucephalus.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/i2y/bucephalus/llm"
	"github.com/i2y/bucephalus/mcp"
	_ "github.com/i2y/bucephalus/openai" // Register OpenAI provider
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Check for MCP server path argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <mcp-server-path> [args...]")
		fmt.Println("\nExample with npx:")
		fmt.Println("  go run main.go npx -y @modelcontextprotocol/server-filesystem /tmp")
		return fmt.Errorf("missing MCP server path argument")
	}

	serverPath := os.Args[1]
	serverArgs := os.Args[2:]

	fmt.Printf("Connecting to MCP server: %s %v\n", serverPath, serverArgs)

	// Get tools from MCP server
	tools, cleanup, err := mcp.ToolsFromMCP(ctx, serverPath, serverArgs)
	if err != nil {
		return fmt.Errorf("connecting to MCP server: %w", err)
	}
	defer func() { _ = cleanup() }()

	fmt.Printf("Discovered %d tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}

	if len(tools) == 0 {
		fmt.Println("No tools available from MCP server")
		return nil
	}

	// Make a call with MCP tools
	fmt.Println("\n--- Making LLM call with MCP tools ---")

	resp, err := llm.Call(ctx, "List the files in the current directory",
		llm.WithProvider("openai"),
		llm.WithModel("o4-mini"),
		llm.WithTools(tools...),
	)
	if err != nil {
		return fmt.Errorf("LLM call: %w", err)
	}

	// Check for tool calls
	if resp.HasToolCalls() {
		fmt.Println("\nModel requested tool calls:")

		// Create a registry for the tools
		registry := llm.NewToolRegistry()
		registry.Register(tools...)

		for _, tc := range resp.ToolCalls() {
			fmt.Printf("  Tool: %s\n", tc.Name)
			fmt.Printf("  Args: %s\n", tc.Arguments)

			// Execute the tool
			tool, ok := registry.Get(tc.Name)
			if !ok {
				fmt.Printf("  Error: tool not found\n")
				continue
			}

			result, err := tool.Execute(ctx, []byte(tc.Arguments))
			if err != nil {
				fmt.Printf("  Error: %v\n", err)
				continue
			}
			fmt.Printf("  Result: %v\n", result)
		}

		// Continue conversation with tool results
		toolMessages, err := llm.ExecuteToolCalls(ctx, resp.ToolCalls(), registry)
		if err != nil {
			return fmt.Errorf("executing tools: %w", err)
		}

		messages := []llm.Message{
			llm.UserMessage("List the files in the current directory"),
			llm.AssistantMessageWithToolCalls("", resp.ToolCalls()),
		}
		messages = append(messages, toolMessages...)

		resp2, err := llm.CallMessages(ctx, messages,
			llm.WithProvider("openai"),
			llm.WithModel("o4-mini"),
		)
		if err != nil {
			return fmt.Errorf("continuing conversation: %w", err)
		}

		fmt.Println("\n--- Final response ---")
		fmt.Println(resp2.Text())
	} else {
		fmt.Println("\nResponse (no tool calls):")
		fmt.Println(resp.Text())
	}

	return nil
}
