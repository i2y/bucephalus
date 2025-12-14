// Package plugin provides support for loading and using Claude Code-style plugins.
package plugin

// Plugin represents a loaded Claude Code-style plugin.
type Plugin struct {
	// Metadata from plugin.json
	Name        string
	Description string
	Version     string
	Author      Author

	// Components
	Commands []Command
	Agents   []Agent
	Skills   []Skill

	// MCP servers configuration
	MCPServers map[string]MCPServerConfig

	// Root path of the plugin
	RootPath string
}

// Author represents plugin author information.
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// Command represents a slash command defined in a plugin.
type Command struct {
	Name        string // Derived from filename (e.g., "hello" from "hello.md")
	Description string // From frontmatter
	Content     string // Markdown content (the prompt)
	FilePath    string // Original file path
}

// Agent represents a subagent defined in a plugin.
type Agent struct {
	Name        string   // Derived from filename
	Description string   // From frontmatter
	Tools       []string // Tools this agent can use
	Content     string   // Markdown content (agent instructions)
	FilePath    string   // Original file path
}

// Skill represents an agent skill defined in a plugin.
type Skill struct {
	Name        string   // Derived from directory name
	Description string   // From frontmatter
	Tools       []string // Tools this skill requires
	Content     string   // Markdown content (skill instructions)
	FilePath    string   // Original file path
}

// MCPServerConfig represents an MCP server configuration.
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// pluginManifest represents the plugin.json structure.
type pluginManifest struct {
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Version     string  `json:"version,omitempty"`
	Author      *Author `json:"author,omitempty"`

	// Custom paths for components
	Commands string `json:"commands,omitempty"`
	Agents   string `json:"agents,omitempty"`
	Skills   string `json:"skills,omitempty"`

	// Inline or path to hooks/mcp config
	Hooks      any `json:"hooks,omitempty"`
	MCPServers any `json:"mcpServers,omitempty"`
}

// commandFrontmatter represents the YAML frontmatter in command files.
type commandFrontmatter struct {
	Description string   `yaml:"description"`
	Allowed     []string `yaml:"allowed,omitempty"` // Allowed tools/contexts
}

// agentFrontmatter represents the YAML frontmatter in agent files.
type agentFrontmatter struct {
	Description string   `yaml:"description"`
	Tools       []string `yaml:"tools,omitempty"`
}

// skillFrontmatter represents the YAML frontmatter in SKILL.md files.
type skillFrontmatter struct {
	Description string   `yaml:"description"`
	Tools       []string `yaml:"tools,omitempty"`
}
