package tools

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/i2y/bucephalus/llm"
)

// GrepInput defines the input for the Grep tool.
type GrepInput struct {
	Pattern    string `json:"pattern" jsonschema:"required,description=Regular expression pattern to search for"`
	Path       string `json:"path,omitempty" jsonschema:"description=File or directory to search in (default: current directory)"`
	Glob       string `json:"glob,omitempty" jsonschema:"description=File pattern filter (e.g. *.go)"`
	MaxMatches int    `json:"max_matches,omitempty" jsonschema:"description=Maximum number of matches to return (default: 100)"`
}

// GrepOutput defines the output of the Grep tool.
type GrepOutput struct {
	Matches []GrepMatch `json:"matches"`
	Count   int         `json:"count"`
}

// GrepMatch represents a single match.
type GrepMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// GrepTool returns the Grep tool.
func GrepTool() (llm.Tool, error) {
	return llm.NewTool(
		"grep",
		"Search for a regular expression pattern in files. Returns matching lines with file and line number.",
		grepFiles,
	)
}

// MustGrep returns the Grep tool, panicking on error.
func MustGrep() llm.Tool {
	tool, err := GrepTool()
	if err != nil {
		panic(err)
	}
	return tool
}

func grepFiles(ctx context.Context, input GrepInput) (GrepOutput, error) {
	re, err := regexp.Compile(input.Pattern)
	if err != nil {
		return GrepOutput{}, err
	}

	basePath := input.Path
	if basePath == "" {
		basePath = "."
	}

	maxMatches := input.MaxMatches
	if maxMatches <= 0 {
		maxMatches = 100
	}

	var matches []GrepMatch

	// Determine files to search
	var files []string

	info, err := os.Stat(basePath)
	if err != nil {
		return GrepOutput{}, err
	}

	if info.IsDir() {
		// Use glob pattern if provided, otherwise search all files
		globPattern := input.Glob
		if globPattern == "" {
			globPattern = "**/*"
		}

		fsys := os.DirFS(basePath)
		globMatches, err := doublestar.Glob(fsys, globPattern)
		if err != nil {
			return GrepOutput{}, err
		}

		for _, m := range globMatches {
			fullPath := filepath.Join(basePath, m)
			finfo, err := os.Stat(fullPath)
			if err == nil && !finfo.IsDir() {
				files = append(files, fullPath)
			}
		}
	} else {
		files = []string{basePath}
	}

	// Search each file
	for _, filePath := range files {
		if len(matches) >= maxMatches {
			break
		}

		fileMatches, err := searchFile(filePath, re, maxMatches-len(matches))
		if err != nil {
			// Skip files that can't be read (e.g., binary files)
			continue
		}
		matches = append(matches, fileMatches...)
	}

	return GrepOutput{
		Matches: matches,
		Count:   len(matches),
	}, nil
}

func searchFile(filePath string, re *regexp.Regexp, maxMatches int) ([]GrepMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var matches []GrepMatch
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if re.MatchString(line) {
			matches = append(matches, GrepMatch{
				File:    filePath,
				Line:    lineNum,
				Content: line,
			})

			if len(matches) >= maxMatches {
				break
			}
		}
	}

	return matches, scanner.Err()
}
