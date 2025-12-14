package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/i2y/bucephalus/llm"
)

// WriteInput defines the input for the Write tool.
type WriteInput struct {
	Path    string `json:"path" jsonschema:"required,description=File path to write"`
	Content string `json:"content" jsonschema:"required,description=Content to write to the file"`
}

// WriteOutput defines the output of the Write tool.
type WriteOutput struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	Bytes   int    `json:"bytes"`
}

// WriteTool returns the Write tool.
func WriteTool() (llm.Tool, error) {
	return llm.NewTool(
		"write",
		"Write content to a file. Creates parent directories if needed.",
		writeFile,
	)
}

// MustWrite returns the Write tool, panicking on error.
func MustWrite() llm.Tool {
	tool, err := WriteTool()
	if err != nil {
		panic(err)
	}
	return tool
}

func writeFile(ctx context.Context, input WriteInput) (WriteOutput, error) {
	// Create parent directories if needed
	dir := filepath.Dir(input.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return WriteOutput{}, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write the file
	data := []byte(input.Content)
	if err := os.WriteFile(input.Path, data, 0o644); err != nil {
		return WriteOutput{}, fmt.Errorf("failed to write file: %w", err)
	}

	return WriteOutput{
		Success: true,
		Path:    input.Path,
		Bytes:   len(data),
	}, nil
}
