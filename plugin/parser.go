package plugin

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseMarkdownWithFrontmatter parses a markdown file and extracts YAML frontmatter.
// Returns the frontmatter bytes and the content after frontmatter.
func parseMarkdownWithFrontmatter(path string) (frontmatter []byte, content string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("reading file: %w", err)
	}

	return parseFrontmatter(data)
}

// parseFrontmatter extracts YAML frontmatter from markdown content.
// Frontmatter is delimited by "---" at the start and end.
func parseFrontmatter(data []byte) (frontmatter []byte, content string, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	// Check for opening delimiter
	if !scanner.Scan() {
		return nil, string(data), nil
	}
	firstLine := strings.TrimSpace(scanner.Text())
	if firstLine != "---" {
		// No frontmatter, return entire content
		return nil, string(data), nil
	}

	// Collect frontmatter lines until closing delimiter
	var fmLines []string
	foundClosing := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundClosing = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if !foundClosing {
		// No closing delimiter, treat as no frontmatter
		return nil, string(data), nil
	}

	// Collect remaining content
	var contentLines []string
	for scanner.Scan() {
		contentLines = append(contentLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("scanning file: %w", err)
	}

	frontmatter = []byte(strings.Join(fmLines, "\n"))
	content = strings.TrimSpace(strings.Join(contentLines, "\n"))

	return frontmatter, content, nil
}

// ParseCommand parses a command markdown file.
func ParseCommand(path string) (*Command, error) {
	fm, content, err := parseMarkdownWithFrontmatter(path)
	if err != nil {
		return nil, fmt.Errorf("parsing command file %s: %w", path, err)
	}

	cmd := &Command{
		Name:     strings.TrimSuffix(filepath.Base(path), ".md"),
		Content:  content,
		FilePath: path,
	}

	if len(fm) > 0 {
		var meta commandFrontmatter
		if err := yaml.Unmarshal(fm, &meta); err != nil {
			return nil, fmt.Errorf("parsing command frontmatter: %w", err)
		}
		cmd.Description = meta.Description
	}

	return cmd, nil
}

// ParseAgent parses an agent markdown file.
func ParseAgent(path string) (*Agent, error) {
	fm, content, err := parseMarkdownWithFrontmatter(path)
	if err != nil {
		return nil, fmt.Errorf("parsing agent file %s: %w", path, err)
	}

	agent := &Agent{
		Name:     strings.TrimSuffix(filepath.Base(path), ".md"),
		Content:  content,
		FilePath: path,
	}

	if len(fm) > 0 {
		var meta agentFrontmatter
		if err := yaml.Unmarshal(fm, &meta); err != nil {
			return nil, fmt.Errorf("parsing agent frontmatter: %w", err)
		}
		agent.Description = meta.Description
		agent.Tools = meta.Tools
	}

	return agent, nil
}

// ParseSkill parses a skill from a directory containing SKILL.md.
func ParseSkill(dirPath string) (*Skill, error) {
	skillFile := filepath.Join(dirPath, "SKILL.md")

	fm, content, err := parseMarkdownWithFrontmatter(skillFile)
	if err != nil {
		return nil, fmt.Errorf("parsing skill file %s: %w", skillFile, err)
	}

	skill := &Skill{
		Name:     filepath.Base(dirPath),
		Content:  content,
		FilePath: skillFile,
	}

	if len(fm) > 0 {
		var meta skillFrontmatter
		if err := yaml.Unmarshal(fm, &meta); err != nil {
			return nil, fmt.Errorf("parsing skill frontmatter: %w", err)
		}
		skill.Description = meta.Description
		skill.Tools = meta.Tools
	}

	return skill, nil
}
