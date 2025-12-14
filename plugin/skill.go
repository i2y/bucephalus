package plugin

import (
	"github.com/i2y/bucephalus/llm"
)

// ToOption converts a Skill to an llm.Option.
// This adds the skill's system message to the LLM call.
func (s *Skill) ToOption() llm.Option {
	return llm.WithSystemMessage(s.ToSystemMessage())
}

// FilterTools filters a list of tools to only include those required by this skill.
// If the skill has no tool requirements, all tools are returned.
// Tools are matched by name (case-insensitive prefix match).
func (s *Skill) FilterTools(tools []llm.Tool) []llm.Tool {
	if len(s.Tools) == 0 {
		return tools
	}

	// Create a set of required tool names
	required := make(map[string]bool)
	for _, name := range s.Tools {
		required[name] = true
	}

	// Filter tools
	var filtered []llm.Tool
	for _, tool := range tools {
		if required[tool.Name()] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// HasRequiredTools checks if all required tools are available in the provided list.
func (s *Skill) HasRequiredTools(tools []llm.Tool) bool {
	if len(s.Tools) == 0 {
		return true
	}

	available := make(map[string]bool)
	for _, tool := range tools {
		available[tool.Name()] = true
	}

	for _, required := range s.Tools {
		if !available[required] {
			return false
		}
	}
	return true
}

// MissingTools returns the list of required tools that are not available.
func (s *Skill) MissingTools(tools []llm.Tool) []string {
	if len(s.Tools) == 0 {
		return nil
	}

	available := make(map[string]bool)
	for _, tool := range tools {
		available[tool.Name()] = true
	}

	var missing []string
	for _, required := range s.Tools {
		if !available[required] {
			missing = append(missing, required)
		}
	}
	return missing
}
