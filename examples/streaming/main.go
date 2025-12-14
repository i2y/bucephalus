// Package main demonstrates streaming with bucephalus.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/i2y/bucephalus/llm"
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

	fmt.Println("Starting streaming response...")

	// CallStream returns a Stream that you can iterate over
	stream, err := llm.CallStream(ctx, "Write a short haiku about programming",
		llm.WithProvider("openai"),
		llm.WithModel("o4-mini"),
	)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()

	// Iterate over chunks using range-over-func (Go 1.23+)
	for chunk := range stream.Chunks() {
		fmt.Print(chunk.Delta)
	}

	// Check for errors after streaming
	if err := stream.Err(); err != nil {
		return fmt.Errorf("stream error: %w", err)
	}

	fmt.Println("\n\n--- Streaming complete ---")

	// Get the accumulated response
	resp := stream.Response()
	fmt.Printf("\nFull response:\n%s\n", resp.Text())
	fmt.Printf("\nUsage: %d total tokens\n", resp.Usage().TotalTokens)

	return nil
}
