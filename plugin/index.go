package plugin

import (
	"fmt"
	"strings"
)

// SkillIndex represents a skill's metadata for progressive disclosure.
// This allows presenting skill information without loading full content.
type SkillIndex struct {
	Name        string
	Description string
}

// CommandIndex represents a command's metadata for progressive disclosure.
type CommandIndex struct {
	Name        string
	Description string
}

// AgentIndex represents an agent's metadata for progressive disclosure.
type AgentIndex struct {
	Name        string
	Description string
	Tools       []string
}

// SkillsIndex returns metadata for all skills in the plugin.
// Use this for progressive disclosure - present the list first,
// then load full skill content only when needed.
func (p *Plugin) SkillsIndex() []SkillIndex {
	result := make([]SkillIndex, len(p.Skills))
	for i, s := range p.Skills {
		result[i] = SkillIndex{
			Name:        s.Name,
			Description: s.Description,
		}
	}
	return result
}

// CommandsIndex returns metadata for all commands in the plugin.
func (p *Plugin) CommandsIndex() []CommandIndex {
	result := make([]CommandIndex, len(p.Commands))
	for i, c := range p.Commands {
		result[i] = CommandIndex{
			Name:        c.Name,
			Description: c.Description,
		}
	}
	return result
}

// AgentsIndex returns metadata for all agents in the plugin.
func (p *Plugin) AgentsIndex() []AgentIndex {
	result := make([]AgentIndex, len(p.Agents))
	for i, a := range p.Agents {
		result[i] = AgentIndex{
			Name:        a.Name,
			Description: a.Description,
			Tools:       a.Tools,
		}
	}
	return result
}

// SkillsIndexSystemMessage returns a compact skills list for system prompt.
// This follows a progressive disclosure pattern (similar to Claude Code) - include only
// metadata in the system prompt, load full content when skill is invoked.
//
// Format:
//
//	<available_skills>
//	- skill-name: Description of the skill
//	</available_skills>
func (p *Plugin) SkillsIndexSystemMessage() string {
	if len(p.Skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")
	for _, s := range p.Skills {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", s.Name, s.Description))
	}
	sb.WriteString("</available_skills>\n\n")
	sb.WriteString("When a skill is relevant to the user's task, mention which skill you would use and why.")

	return sb.String()
}

// CommandsIndexSystemMessage returns a compact commands list for system prompt.
//
// Format:
//
//	<available_commands>
//	- /command-name: Description of the command
//	</available_commands>
func (p *Plugin) CommandsIndexSystemMessage() string {
	if len(p.Commands) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_commands>\n")
	for _, c := range p.Commands {
		sb.WriteString(fmt.Sprintf("- /%s: %s\n", c.Name, c.Description))
	}
	sb.WriteString("</available_commands>\n\n")
	sb.WriteString("Users can invoke these commands by typing /<command-name> followed by any arguments.")

	return sb.String()
}

// AgentsIndexSystemMessage returns a compact agents list for system prompt.
//
// Format:
//
//	<available_agents>
//	- agent-name: Description (tools: tool1, tool2)
//	</available_agents>
func (p *Plugin) AgentsIndexSystemMessage() string {
	if len(p.Agents) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_agents>\n")
	for _, a := range p.Agents {
		if len(a.Tools) > 0 {
			sb.WriteString(fmt.Sprintf("- %s: %s (tools: %s)\n",
				a.Name, a.Description, strings.Join(a.Tools, ", ")))
		} else {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", a.Name, a.Description))
		}
	}
	sb.WriteString("</available_agents>\n\n")
	sb.WriteString("Agents can be spawned to handle specific tasks independently.")

	return sb.String()
}

// PluginIndexSystemMessage returns a combined index of all plugin components.
// This is useful for giving the LLM an overview of available capabilities.
func (p *Plugin) PluginIndexSystemMessage() string {
	var parts []string

	if msg := p.SkillsIndexSystemMessage(); msg != "" {
		parts = append(parts, msg)
	}
	if msg := p.CommandsIndexSystemMessage(); msg != "" {
		parts = append(parts, msg)
	}
	if msg := p.AgentsIndexSystemMessage(); msg != "" {
		parts = append(parts, msg)
	}

	if len(parts) == 0 {
		return ""
	}

	header := fmt.Sprintf("# Plugin: %s\n\n", p.Name)
	if p.Description != "" {
		header += p.Description + "\n\n"
	}

	return header + strings.Join(parts, "\n")
}

// HasSkill checks if a skill with the given name exists.
func (p *Plugin) HasSkill(name string) bool {
	return p.GetSkill(name) != nil
}

// HasCommand checks if a command with the given name exists.
func (p *Plugin) HasCommand(name string) bool {
	return p.GetCommand(name) != nil
}

// HasAgent checks if an agent with the given name exists.
func (p *Plugin) HasAgent(name string) bool {
	return p.GetAgent(name) != nil
}
