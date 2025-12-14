package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantFrontmatter string
		wantContent     string
	}{
		{
			name: "valid frontmatter",
			input: `---
description: Test command
---
This is the content.`,
			wantFrontmatter: "description: Test command",
			wantContent:     "This is the content.",
		},
		{
			name:            "no frontmatter",
			input:           "Just content without frontmatter.",
			wantFrontmatter: "",
			wantContent:     "Just content without frontmatter.",
		},
		{
			name: "frontmatter without closing delimiter",
			input: `---
description: Test
This is content`,
			wantFrontmatter: "",
			wantContent: `---
description: Test
This is content`,
		},
		{
			name:            "empty input",
			input:           "",
			wantFrontmatter: "",
			wantContent:     "",
		},
		{
			name: "frontmatter with multiple fields",
			input: `---
description: Multi-field
tools:
  - tool1
  - tool2
---
Content after frontmatter.`,
			wantFrontmatter: "description: Multi-field\ntools:\n  - tool1\n  - tool2",
			wantContent:     "Content after frontmatter.",
		},
		{
			name: "empty frontmatter",
			input: `---
---
Content only.`,
			wantFrontmatter: "",
			wantContent:     "Content only.",
		},
		{
			name: "multiline content",
			input: `---
description: Test
---
Line 1
Line 2
Line 3`,
			wantFrontmatter: "description: Test",
			wantContent:     "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, content, err := parseFrontmatter([]byte(tt.input))

			require.NoError(t, err)
			assert.Equal(t, tt.wantFrontmatter, string(fm))
			assert.Equal(t, tt.wantContent, content)
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     string
		wantName    string
		wantDesc    string
		wantContent string
	}{
		{
			name:     "command with frontmatter",
			filename: "greet.md",
			content: `---
description: Greet someone
---
Hello, $ARGUMENTS!`,
			wantName:    "greet",
			wantDesc:    "Greet someone",
			wantContent: "Hello, $ARGUMENTS!",
		},
		{
			name:        "command without frontmatter",
			filename:    "simple.md",
			content:     "Just say hello.",
			wantName:    "simple",
			wantDesc:    "",
			wantContent: "Just say hello.",
		},
		{
			name:     "command with long description",
			filename: "complex.md",
			content: `---
description: This is a more complex command that does many things
---
Perform the following steps:
1. First step
2. Second step`,
			wantName:    "complex",
			wantDesc:    "This is a more complex command that does many things",
			wantContent: "Perform the following steps:\n1. First step\n2. Second step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(path, []byte(tt.content), 0o644)
			require.NoError(t, err)

			cmd, err := ParseCommand(path)

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, cmd.Name)
			assert.Equal(t, tt.wantDesc, cmd.Description)
			assert.Equal(t, tt.wantContent, cmd.Content)
			assert.Equal(t, path, cmd.FilePath)
		})
	}
}

func TestParseCommand_FileNotFound(t *testing.T) {
	_, err := ParseCommand("/nonexistent/path/command.md")
	assert.Error(t, err)
}

func TestParseAgent(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     string
		wantName    string
		wantDesc    string
		wantTools   []string
		wantContent string
	}{
		{
			name:     "agent with tools",
			filename: "helper.md",
			content: `---
description: A helpful agent
tools:
  - read
  - write
---
You are a helpful agent.`,
			wantName:    "helper",
			wantDesc:    "A helpful agent",
			wantTools:   []string{"read", "write"},
			wantContent: "You are a helpful agent.",
		},
		{
			name:        "agent without frontmatter",
			filename:    "simple.md",
			content:     "A simple agent.",
			wantName:    "simple",
			wantDesc:    "",
			wantTools:   nil,
			wantContent: "A simple agent.",
		},
		{
			name:     "agent with description only",
			filename: "desc-only.md",
			content: `---
description: Agent with description only
---
Do the task.`,
			wantName:    "desc-only",
			wantDesc:    "Agent with description only",
			wantTools:   nil,
			wantContent: "Do the task.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(path, []byte(tt.content), 0o644)
			require.NoError(t, err)

			agent, err := ParseAgent(path)

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, agent.Name)
			assert.Equal(t, tt.wantDesc, agent.Description)
			assert.Equal(t, tt.wantTools, agent.Tools)
			assert.Equal(t, tt.wantContent, agent.Content)
			assert.Equal(t, path, agent.FilePath)
		})
	}
}

func TestParseSkill(t *testing.T) {
	tests := []struct {
		name        string
		dirName     string
		content     string
		wantName    string
		wantDesc    string
		wantTools   []string
		wantContent string
	}{
		{
			name:    "skill with tools",
			dirName: "code-review",
			content: `---
description: Code review skill
tools:
  - read
  - grep
---
Review the code for issues.`,
			wantName:    "code-review",
			wantDesc:    "Code review skill",
			wantTools:   []string{"read", "grep"},
			wantContent: "Review the code for issues.",
		},
		{
			name:        "skill without frontmatter",
			dirName:     "simple-skill",
			content:     "A simple skill.",
			wantName:    "simple-skill",
			wantDesc:    "",
			wantTools:   nil,
			wantContent: "A simple skill.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			skillDir := filepath.Join(tmpDir, tt.dirName)
			err := os.MkdirAll(skillDir, 0o755)
			require.NoError(t, err)

			skillFile := filepath.Join(skillDir, "SKILL.md")
			err = os.WriteFile(skillFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			skill, err := ParseSkill(skillDir)

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, skill.Name)
			assert.Equal(t, tt.wantDesc, skill.Description)
			assert.Equal(t, tt.wantTools, skill.Tools)
			assert.Equal(t, tt.wantContent, skill.Content)
			assert.Equal(t, skillFile, skill.FilePath)
		})
	}
}

func TestParseSkill_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "empty-skill")
	err := os.MkdirAll(skillDir, 0o755)
	require.NoError(t, err)

	_, err = ParseSkill(skillDir)
	assert.Error(t, err)
}
