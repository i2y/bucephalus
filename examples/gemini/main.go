// Package main demonstrates the Gemini provider.
package main

import (
	"context"
	"fmt"
	"os"

	_ "github.com/i2y/bucephalus/gemini" // Register Gemini provider
	"github.com/i2y/bucephalus/llm"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Gemini Provider Demo ===")
	fmt.Println()

	// Simple call
	resp, err := llm.Call(ctx, "What is Go programming language? Answer in 2 sentences.",
		llm.WithProvider("gemini"),
		llm.WithModel("gemini-2.5-flash"),
		llm.WithMaxTokens(256),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Question: What is Go programming language?")
	fmt.Printf("Answer: %s\n\n", resp.Text())

	// With system message
	resp2, err := llm.Call(ctx, "Translate 'Hello, World!' to Japanese",
		llm.WithProvider("gemini"),
		llm.WithModel("gemini-2.5-flash"),
		llm.WithSystemMessage("You are a helpful translator. Respond with only the translation."),
		llm.WithMaxTokens(100),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Question: Translate 'Hello, World!' to Japanese")
	fmt.Printf("Answer: %s\n\n", resp2.Text())

	// Multi-turn conversation with Resume
	fmt.Println("=== Multi-turn Conversation ===")

	resp3, err := llm.Call(ctx, "What is 2 + 2?",
		llm.WithProvider("gemini"),
		llm.WithModel("gemini-2.5-flash"),
		llm.WithMaxTokens(100),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User: What is 2 + 2?\n")
	fmt.Printf("Gemini: %s\n\n", resp3.Text())

	resp4, err := resp3.Resume(ctx, "And what is that multiplied by 3?")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User: And what is that multiplied by 3?\n")
	fmt.Printf("Gemini: %s\n", resp4.Text())
}
