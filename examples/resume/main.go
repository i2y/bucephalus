// Package main demonstrates the Resume functionality for multi-turn conversations.
package main

import (
	"context"
	"fmt"
	"os"

	_ "github.com/i2y/bucephalus/anthropic" // Register Anthropic provider
	"github.com/i2y/bucephalus/llm"
)

func main() {
	ctx := context.Background()

	// Start a conversation
	fmt.Println("=== Multi-turn Conversation with Resume ===")
	fmt.Println()

	resp1, err := llm.Call(ctx, "Recommend a fantasy book",
		llm.WithProvider("anthropic"),
		llm.WithModel("claude-3-5-haiku-latest"),
		llm.WithMaxTokens(256),
		llm.WithSystemMessage("You are a helpful librarian. Keep your responses concise."),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("User: Recommend a fantasy book")
	fmt.Printf("Assistant: %s\n\n", resp1.Text())

	// Continue the conversation using Resume
	resp2, err := resp1.Resume(ctx, "Why did you recommend that one?")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("User: Why did you recommend that one?")
	fmt.Printf("Assistant: %s\n\n", resp2.Text())

	// Continue further
	resp3, err := resp2.Resume(ctx, "What other books by the same author would you recommend?")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("User: What other books by the same author would you recommend?")
	fmt.Printf("Assistant: %s\n\n", resp3.Text())

	// Show the full message history
	fmt.Println("=== Full Message History ===")
	for i, msg := range resp3.Messages() {
		role := string(msg.Role)
		content := msg.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		fmt.Printf("%d. [%s] %s\n", i+1, role, content)
	}
}
