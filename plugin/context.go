package plugin

import (
	"fmt"
	"strings"
)

// ToSystemMessage converts a Command to a system message string.
// This includes the command description and instructions.
func (c *Command) ToSystemMessage() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Command: /%s\n\n", c.Name))

	if c.Description != "" {
		sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", c.Description))
	}

	if c.Content != "" {
		sb.WriteString("**Instructions:**\n\n")
		sb.WriteString(c.Content)
	}

	return sb.String()
}

// ToSystemMessage converts an Agent to a system message string.
// This includes the agent's role, capabilities, and instructions.
func (a *Agent) ToSystemMessage() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Agent: %s\n\n", a.Name))

	if a.Description != "" {
		sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", a.Description))
	}

	if len(a.Tools) > 0 {
		sb.WriteString(fmt.Sprintf("**Available Tools:** %s\n\n", strings.Join(a.Tools, ", ")))
	}

	if a.Content != "" {
		sb.WriteString("**Instructions:**\n\n")
		sb.WriteString(a.Content)
	}

	return sb.String()
}

// ToSystemMessage converts a Skill to a system message string.
// This includes the skill's purpose and instructions.
func (s *Skill) ToSystemMessage() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Skill: %s\n\n", s.Name))

	if s.Description != "" {
		sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", s.Description))
	}

	if len(s.Tools) > 0 {
		sb.WriteString(fmt.Sprintf("**Required Tools:** %s\n\n", strings.Join(s.Tools, ", ")))
	}

	if s.Content != "" {
		sb.WriteString("**Instructions:**\n\n")
		sb.WriteString(s.Content)
	}

	return sb.String()
}

// ToSystemMessage converts the entire Plugin to a comprehensive system message.
// This includes all commands, agents, and skills defined in the plugin.
func (p *Plugin) ToSystemMessage() string {
	var sb strings.Builder

	// Plugin header
	sb.WriteString(fmt.Sprintf("# Plugin: %s\n\n", p.Name))

	if p.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", p.Description))
	}

	// Commands section
	if len(p.Commands) > 0 {
		sb.WriteString("---\n\n# Available Commands\n\n")
		for _, cmd := range p.Commands {
			sb.WriteString(cmd.ToSystemMessage())
			sb.WriteString("\n\n")
		}
	}

	// Agents section
	if len(p.Agents) > 0 {
		sb.WriteString("---\n\n# Available Agents\n\n")
		for _, agent := range p.Agents {
			sb.WriteString(agent.ToSystemMessage())
			sb.WriteString("\n\n")
		}
	}

	// Skills section
	if len(p.Skills) > 0 {
		sb.WriteString("---\n\n# Available Skills\n\n")
		for _, skill := range p.Skills {
			sb.WriteString(skill.ToSystemMessage())
			sb.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(sb.String())
}

// CommandsSystemMessage returns a system message with only the commands.
func (p *Plugin) CommandsSystemMessage() string {
	if len(p.Commands) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Available Commands\n\n")
	for _, cmd := range p.Commands {
		sb.WriteString(cmd.ToSystemMessage())
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}

// AgentsSystemMessage returns a system message with only the agents.
func (p *Plugin) AgentsSystemMessage() string {
	if len(p.Agents) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Available Agents\n\n")
	for _, agent := range p.Agents {
		sb.WriteString(agent.ToSystemMessage())
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}

// SkillsSystemMessage returns a system message with only the skills.
func (p *Plugin) SkillsSystemMessage() string {
	if len(p.Skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Available Skills\n\n")
	for _, skill := range p.Skills {
		sb.WriteString(skill.ToSystemMessage())
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}
