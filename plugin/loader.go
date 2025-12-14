package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Load loads a Claude Code-style plugin from the given path.
// The path should point to the plugin root directory containing .claude-plugin/plugin.json.
func Load(path string) (*Plugin, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("accessing plugin path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("plugin path must be a directory: %s", absPath)
	}

	// Load plugin manifest
	manifestPath := filepath.Join(absPath, ".claude-plugin", "plugin.json")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	plugin := &Plugin{
		Name:        manifest.Name,
		Description: manifest.Description,
		Version:     manifest.Version,
		RootPath:    absPath,
		MCPServers:  make(map[string]MCPServerConfig),
	}

	if manifest.Author != nil {
		plugin.Author = *manifest.Author
	}

	// Load commands
	commandsDir := filepath.Join(absPath, "commands")
	if manifest.Commands != "" {
		commandsDir = filepath.Join(absPath, manifest.Commands)
	}
	if commands, err := loadCommands(commandsDir); err == nil {
		plugin.Commands = commands
	}

	// Load agents
	agentsDir := filepath.Join(absPath, "agents")
	if manifest.Agents != "" {
		agentsDir = filepath.Join(absPath, manifest.Agents)
	}
	if agents, err := loadAgents(agentsDir); err == nil {
		plugin.Agents = agents
	}

	// Load skills
	skillsDir := filepath.Join(absPath, "skills")
	if manifest.Skills != "" {
		skillsDir = filepath.Join(absPath, manifest.Skills)
	}
	if skills, err := loadSkills(skillsDir); err == nil {
		plugin.Skills = skills
	}

	// Load MCP servers
	mcpPath := filepath.Join(absPath, ".mcp.json")
	if servers, err := loadMCPServers(mcpPath, absPath); err == nil {
		plugin.MCPServers = servers
	}

	return plugin, nil
}

// loadManifest loads the plugin.json manifest file.
func loadManifest(path string) (*pluginManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest pluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	if manifest.Name == "" {
		return nil, fmt.Errorf("plugin name is required in manifest")
	}

	return &manifest, nil
}

// loadCommands loads all command files from a directory.
func loadCommands(dir string) ([]Command, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	commands := make([]Command, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		cmd, err := ParseCommand(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue // Skip files that can't be parsed
		}
		commands = append(commands, *cmd)
	}

	return commands, nil
}

// loadAgents loads all agent files from a directory.
func loadAgents(dir string) ([]Agent, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	agents := make([]Agent, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		agent, err := ParseAgent(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue // Skip files that can't be parsed
		}
		agents = append(agents, *agent)
	}

	return agents, nil
}

// loadSkills loads all skills from a directory.
// Each subdirectory containing a SKILL.md file is a skill.
func loadSkills(dir string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	skills := make([]Skill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(dir, entry.Name())
		skillFile := filepath.Join(skillPath, "SKILL.md")

		// Check if SKILL.md exists
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}

		skill, err := ParseSkill(skillPath)
		if err != nil {
			continue // Skip skills that can't be parsed
		}
		skills = append(skills, *skill)
	}

	return skills, nil
}

// loadMCPServers loads MCP server configurations from .mcp.json.
func loadMCPServers(path, pluginRoot string) (map[string]MCPServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing MCP config: %w", err)
	}

	// Replace ${CLAUDE_PLUGIN_ROOT} with actual path
	result := make(map[string]MCPServerConfig)
	for name, cfg := range raw.MCPServers {
		cfg.Command = expandPluginRoot(cfg.Command, pluginRoot)
		for i, arg := range cfg.Args {
			cfg.Args[i] = expandPluginRoot(arg, pluginRoot)
		}
		for k, v := range cfg.Env {
			cfg.Env[k] = expandPluginRoot(v, pluginRoot)
		}
		result[name] = cfg
	}

	return result, nil
}

// expandPluginRoot replaces ${CLAUDE_PLUGIN_ROOT} with the actual plugin root path.
func expandPluginRoot(s, pluginRoot string) string {
	return strings.ReplaceAll(s, "${CLAUDE_PLUGIN_ROOT}", pluginRoot)
}

// GetCommand returns a command by name, or nil if not found.
func (p *Plugin) GetCommand(name string) *Command {
	for i := range p.Commands {
		if p.Commands[i].Name == name {
			return &p.Commands[i]
		}
	}
	return nil
}

// GetAgent returns an agent by name, or nil if not found.
func (p *Plugin) GetAgent(name string) *Agent {
	for i := range p.Agents {
		if p.Agents[i].Name == name {
			return &p.Agents[i]
		}
	}
	return nil
}

// GetSkill returns a skill by name, or nil if not found.
func (p *Plugin) GetSkill(name string) *Skill {
	for i := range p.Skills {
		if p.Skills[i].Name == name {
			return &p.Skills[i]
		}
	}
	return nil
}
