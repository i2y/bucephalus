// Package main demonstrates a simple LLM call with Bucephalus.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/i2y/bucephalus/llm"
	_ "github.com/i2y/bucephalus/openai" // Register OpenAI provider
)

func main() {
	ctx := context.Background()

	// Simple call with direct options
	resp, err := llm.Call(ctx, "Recommend a fantasy book in one sentence",
		llm.WithProvider("openai"),
		llm.WithModel("gpt-4o-mini"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Response:")
	fmt.Println(resp.Text())
	fmt.Printf("\nUsage: %d prompt + %d completion = %d total tokens\n",
		resp.Usage().PromptTokens,
		resp.Usage().CompletionTokens,
		resp.Usage().TotalTokens,
	)

	// Using a reusable Model instance
	fmt.Println("\n--- Using Model instance ---")
	model := llm.NewModel("openai", "gpt-4o-mini",
		llm.WithTemperature(0.7),
	)

	resp2, err := model.Call(ctx, "Tell me a short joke")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Response:")
	fmt.Println(resp2.Text())
}
