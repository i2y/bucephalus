// Package main demonstrates structured output with Bucephalus.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/i2y/bucephalus/llm"
	_ "github.com/i2y/bucephalus/openai" // Register OpenAI provider
)

// Book represents a book recommendation.
type Book struct {
	Title       string   `json:"title" jsonschema:"required,description=The book's title"`
	Author      string   `json:"author" jsonschema:"required,description=The book's author"`
	Year        int      `json:"year,omitempty" jsonschema:"description=Publication year"`
	Genres      []string `json:"genres" jsonschema:"required,description=List of genres"`
	Description string   `json:"description" jsonschema:"required,description=A brief description of the book"`
}

func main() {
	ctx := context.Background()

	// CallParse automatically generates JSON schema from the Book type
	// and parses the response into a Book struct
	resp, err := llm.CallParse[Book](ctx, "Recommend a classic science fiction book",
		llm.WithProvider("openai"),
		llm.WithModel("gpt-4o-mini"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get the parsed book (type-safe!)
	book := resp.MustParse()

	fmt.Println("Book Recommendation:")
	fmt.Printf("  Title:       %s\n", book.Title)
	fmt.Printf("  Author:      %s\n", book.Author)
	if book.Year > 0 {
		fmt.Printf("  Year:        %d\n", book.Year)
	}
	fmt.Printf("  Genres:      %v\n", book.Genres)
	fmt.Printf("  Description: %s\n", book.Description)

	fmt.Printf("\nRaw JSON response:\n%s\n", resp.Text())
}
