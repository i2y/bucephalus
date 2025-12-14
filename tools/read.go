// Package tools provides built-in tools for LLM agents.
package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/i2y/bucephalus/llm"
)

// ReadInput defines the input for the Read tool.
type ReadInput struct {
	Path   string `json:"path" jsonschema:"required,description=File path to read"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=Line offset to start from (0-based)"`
	Limit  int    `json:"limit,omitempty" jsonschema:"description=Max lines to read (default: 0 = all)"`
}

// ReadOutput defines the output of the Read tool.
type ReadOutput struct {
	Content   string `json:"content"`
	Lines     int    `json:"lines"`
	Truncated bool   `json:"truncated"`
}

// ReadTool returns the Read tool.
func ReadTool() (llm.Tool, error) {
	return llm.NewTool(
		"read",
		"Read the contents of a file. Supports reading specific line ranges.",
		readFile,
	)
}

// MustRead returns the Read tool, panicking on error.
func MustRead() llm.Tool {
	tool, err := ReadTool()
	if err != nil {
		panic(err)
	}
	return tool
}

func readFile(ctx context.Context, input ReadInput) (ReadOutput, error) {
	file, err := os.Open(input.Path)
	if err != nil {
		return ReadOutput{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 0
	truncated := false

	for scanner.Scan() {
		// Skip lines before offset
		if lineNum < input.Offset {
			lineNum++
			continue
		}

		// Check limit
		if input.Limit > 0 && len(lines) >= input.Limit {
			truncated = true
			break
		}

		lines = append(lines, scanner.Text())
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return ReadOutput{}, fmt.Errorf("failed to read file: %w", err)
	}

	return ReadOutput{
		Content:   strings.Join(lines, "\n"),
		Lines:     len(lines),
		Truncated: truncated,
	}, nil
}
