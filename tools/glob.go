package tools

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/i2y/bucephalus/llm"
)

// GlobInput defines the input for the Glob tool.
type GlobInput struct {
	Pattern string `json:"pattern" jsonschema:"required,description=Glob pattern (e.g. **/*.go for all Go files)"`
	Path    string `json:"path,omitempty" jsonschema:"description=Base directory to search from (default: current directory)"`
}

// GlobOutput defines the output of the Glob tool.
type GlobOutput struct {
	Files []string `json:"files"`
	Count int      `json:"count"`
}

// GlobTool returns the Glob tool.
func GlobTool() (llm.Tool, error) {
	return llm.NewTool(
		"glob",
		"Find files matching a glob pattern. Supports ** for recursive matching.",
		globFiles,
	)
}

// MustGlob returns the Glob tool, panicking on error.
func MustGlob() llm.Tool {
	tool, err := GlobTool()
	if err != nil {
		panic(err)
	}
	return tool
}

func globFiles(ctx context.Context, input GlobInput) (GlobOutput, error) {
	basePath := input.Path
	if basePath == "" {
		basePath = "."
	}

	// Clean and normalize the base path
	basePath = filepath.Clean(basePath)

	// Use the base path as the filesystem root
	fsys := os.DirFS(basePath)
	matches, err := doublestar.Glob(fsys, input.Pattern)
	if err != nil {
		return GlobOutput{}, err
	}

	// Prepend base path to results if not current directory
	if basePath != "." {
		for i, m := range matches {
			matches[i] = filepath.Join(basePath, m)
		}
	}

	return GlobOutput{
		Files: matches,
		Count: len(matches),
	}, nil
}
