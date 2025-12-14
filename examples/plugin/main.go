package main

import (
	"context"
	"fmt"
	"log"
	"os"

	_ "github.com/i2y/bucephalus/anthropic" // Register Anthropic provider
	"github.com/i2y/bucephalus/llm"
	"github.com/i2y/bucephalus/plugin"
)

func main() {
	// Load the test plugin
	p, err := plugin.Load("./test-plugin")
	if err != nil {
		log.Fatalf("Failed to load plugin: %v", err)
	}

	fmt.Println("=== Plugin Loaded ===")
	fmt.Printf("Name: %s\n", p.Name)
	fmt.Printf("Description: %s\n", p.Description)
	fmt.Printf("Version: %s\n", p.Version)
	fmt.Println()

	// Display commands
	fmt.Println("=== Commands ===")
	for _, cmd := range p.Commands {
		fmt.Printf("- /%s: %s\n", cmd.Name, cmd.Description)
	}
	fmt.Println()

	// Display agents
	fmt.Println("=== Agents ===")
	for _, agent := range p.Agents {
		fmt.Printf("- %s: %s (tools: %v)\n", agent.Name, agent.Description, agent.Tools)
	}
	fmt.Println()

	// Display skills
	fmt.Println("=== Skills ===")
	for _, skill := range p.Skills {
		fmt.Printf("- %s: %s (tools: %v)\n", skill.Name, skill.Description, skill.Tools)
	}
	fmt.Println()

	// Check if API key is available for LLM demos
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Println("Set ANTHROPIC_API_KEY to test LLM integration")
		fmt.Println()
		fmt.Println("=== Demo: Command Expansion (without LLM) ===")
		demoCommandExpansion(p)
		return
	}

	ctx := context.Background()

	// Demo 1: Command Expansion
	fmt.Println("=== Demo 1: Command Expansion ===")
	demoCommandWithLLM(ctx, p)

	// Demo 2: Command with Arguments ($ARGUMENTS)
	fmt.Println("\n=== Demo 2: Command with $ARGUMENTS ===")
	demoCommandWithArgs(ctx, p)

	// Demo 3: ProcessInput (auto-detect command)
	fmt.Println("\n=== Demo 3: ProcessInput (Auto-detect) ===")
	demoProcessInput(ctx, p)

	// Demo 4: Skill Usage
	fmt.Println("\n=== Demo 4: Skill Usage ===")
	demoSkill(ctx, p)

	// Demo 5: Agent Runner
	fmt.Println("\n=== Demo 5: Agent Runner ===")
	demoAgentRunner(ctx, p)

	// Demo 6: Agent Context (Multi-turn Conversation)
	fmt.Println("\n=== Demo 6: Agent Context (Multi-turn) ===")
	demoAgentContext(ctx, p)

	// Demo 7: Progressive Disclosure (Index System Messages)
	fmt.Println("\n=== Demo 7: Progressive Disclosure ===")
	demoProgressiveDisclosure(p)
}

func demoCommandExpansion(p *plugin.Plugin) {
	// Test command detection
	inputs := []string{
		"/greet",
		"/translate Hello World",
		"regular message",
	}

	for _, input := range inputs {
		fmt.Printf("Input: %q\n", input)
		fmt.Printf("  IsCommand: %v\n", p.IsCommand(input))

		if p.IsCommand(input) {
			expanded, err := p.ExpandCommand(input)
			if err != nil {
				fmt.Printf("  Error: %v\n", err)
			} else {
				fmt.Printf("  Command: %s\n", expanded.Command.Name)
				fmt.Printf("  Arguments: %q\n", expanded.Arguments)
				fmt.Printf("  SystemMessage (first 100 chars): %s...\n",
					truncate(expanded.SystemMessage, 100))
			}
		}
		fmt.Println()
	}
}

func demoCommandWithLLM(ctx context.Context, p *plugin.Plugin) {
	// Use greet command
	greetCmd := p.GetCommand("greet")
	if greetCmd == nil {
		fmt.Println("greet command not found")
		return
	}

	resp, err := llm.Call(ctx, "Hello!",
		llm.WithProvider("anthropic"),
		llm.WithModel("claude-3-5-haiku-latest"),
		greetCmd.ToOption(), // Use command as system message
		llm.WithMaxTokens(200),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Command: /greet\n")
	fmt.Printf("Response: %s\n", resp.Text())
}

func demoCommandWithArgs(ctx context.Context, p *plugin.Plugin) {
	// Expand translate command with arguments
	expanded, err := p.ExpandCommand("/translate Hello, how are you?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Command: /translate Hello, how are you?\n")
	fmt.Printf("Expanded SystemMessage:\n%s\n\n", expanded.SystemMessage)

	// Use the arguments as user message (or "Please proceed" if empty)
	userMsg := expanded.Arguments
	if userMsg == "" {
		userMsg = "Please proceed with the task."
	}

	resp, err := llm.Call(ctx, userMsg,
		llm.WithProvider("anthropic"),
		llm.WithModel("claude-3-5-haiku-latest"),
		expanded.ToOption(),
		llm.WithMaxTokens(100),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.Text())
}

func demoProcessInput(ctx context.Context, p *plugin.Plugin) {
	inputs := []string{
		"/greet",
		"What is Go programming language?",
	}

	for _, input := range inputs {
		fmt.Printf("Input: %q\n", input)

		opt, userMsg, err := p.ProcessInput(input)
		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		opts := []llm.Option{
			llm.WithProvider("anthropic"),
			llm.WithModel("claude-3-5-haiku-latest"),
			llm.WithMaxTokens(150),
		}
		if opt != nil {
			opts = append(opts, opt)
			fmt.Printf("  → Command detected, using expanded system message\n")
		} else {
			fmt.Printf("  → Regular message\n")
		}

		resp, err := llm.Call(ctx, userMsg, opts...)
		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		fmt.Printf("Response: %s\n\n", truncate(resp.Text(), 200))
	}
}

func demoSkill(ctx context.Context, p *plugin.Plugin) {
	skill := p.GetSkill("code-review")
	if skill == nil {
		fmt.Println("code-review skill not found")
		return
	}

	code := `func add(a, b int) int {
    return a + b
}`

	resp, err := llm.Call(ctx, "Review this Go function:\n"+code,
		llm.WithProvider("anthropic"),
		llm.WithModel("claude-3-5-haiku-latest"),
		skill.ToOption(), // Use skill's system message
		llm.WithMaxTokens(300),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Skill: %s\n", skill.Name)
	fmt.Printf("Code:\n%s\n\n", code)
	fmt.Printf("Review:\n%s\n", resp.Text())
}

func demoAgentRunner(ctx context.Context, p *plugin.Plugin) {
	agent := p.GetAgent("helper")
	if agent == nil {
		fmt.Println("helper agent not found")
		return
	}

	// Create a runner for the agent
	runner := agent.NewRunner(
		plugin.WithAgentProvider("anthropic"),
		plugin.WithAgentModel("claude-3-5-haiku-latest"),
		plugin.WithAgentMaxTokens(200),
	)

	fmt.Printf("Agent: %s\n", agent.Name)
	fmt.Printf("Description: %s\n", agent.Description)
	fmt.Printf("Allowed Tools: %v\n\n", agent.Tools)

	// Run the agent with a task
	resp, err := runner.Run(ctx, "Explain what you can help me with in one sentence.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Task: Explain what you can help me with in one sentence.\n")
	fmt.Printf("Response: %s\n", resp.Text())
}

func demoAgentContext(ctx context.Context, p *plugin.Plugin) {
	agent := p.GetAgent("helper")
	if agent == nil {
		fmt.Println("helper agent not found")
		return
	}

	// Create a runner with extra LLM options (案1: NewRunner時)
	// WithAgentLLMOptions allows passing any llm.Option
	runner := agent.NewRunner(
		plugin.WithAgentProvider("anthropic"),
		plugin.WithAgentModel("claude-3-5-haiku-latest"),
		plugin.WithAgentMaxTokens(150),
		// NEW: Pass additional llm.Options at runner creation
		plugin.WithAgentLLMOptions(
			llm.WithSystemMessage(p.SkillsIndexSystemMessage()), // Add skills index
		),
	)

	fmt.Printf("Agent: %s\n", agent.Name)
	fmt.Printf("Initial history length: %d\n\n", runner.Context().HistoryLen())

	// First conversation turn
	fmt.Println("--- Turn 1 ---")
	resp1, err := runner.Run(ctx, "My name is Alice. Please remember it.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("User: My name is Alice. Please remember it.\n")
	fmt.Printf("Assistant: %s\n", truncate(resp1.Text(), 150))
	fmt.Printf("History length after turn 1: %d\n\n", runner.Context().HistoryLen())

	// Second conversation turn - context is maintained
	fmt.Println("--- Turn 2 ---")
	resp2, err := runner.Run(ctx, "What is my name?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("User: What is my name?\n")
	fmt.Printf("Assistant: %s\n", truncate(resp2.Text(), 150))
	fmt.Printf("History length after turn 2: %d\n\n", runner.Context().HistoryLen())

	// Demonstrate state storage
	fmt.Println("--- State Storage ---")
	runner.Context().SetState("user_preference", "dark_mode")
	runner.Context().SetState("session_id", 12345)

	if pref, ok := runner.Context().GetState("user_preference"); ok {
		fmt.Printf("Stored state - user_preference: %v\n", pref)
	}
	if sid, ok := runner.Context().GetState("session_id"); ok {
		fmt.Printf("Stored state - session_id: %v\n", sid)
	}

	// Show conversation history
	fmt.Println("\n--- Full Conversation History ---")
	for i, msg := range runner.Context().History() {
		role := "unknown"
		switch msg.Role {
		case "user":
			role = "User"
		case "assistant":
			role = "Assistant"
		}
		fmt.Printf("%d. [%s]: %s\n", i+1, role, truncate(msg.Content, 50))
	}

	// Clear and demonstrate fresh start
	fmt.Println("\n--- After ClearHistory ---")
	runner.ClearHistory()
	fmt.Printf("History length: %d\n", runner.Context().HistoryLen())

	// State is preserved even after ClearHistory
	if pref, ok := runner.Context().GetState("user_preference"); ok {
		fmt.Printf("State preserved - user_preference: %v\n", pref)
	}

	// Demonstrate Run-time options (案2: Run時)
	fmt.Println("\n--- Run-time Options Demo ---")
	resp3, err := runner.Run(ctx, "What skills are available?",
		// NEW: Pass options for this specific Run() call only
		plugin.WithRunSystemMessage("Focus on listing available skills briefly."),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("With RunOption: %s\n", truncate(resp3.Text(), 150))
}

func demoProgressiveDisclosure(p *plugin.Plugin) {
	// Show how progressive disclosure works:
	// Instead of including full skill/command content in system prompt,
	// only include metadata (name, description) to save context tokens.

	fmt.Println("--- Skills Index ---")
	for _, s := range p.SkillsIndex() {
		fmt.Printf("- %s: %s\n", s.Name, s.Description)
	}

	fmt.Println("\n--- Commands Index ---")
	for _, c := range p.CommandsIndex() {
		fmt.Printf("- /%s: %s\n", c.Name, c.Description)
	}

	fmt.Println("\n--- Agents Index ---")
	for _, a := range p.AgentsIndex() {
		if len(a.Tools) > 0 {
			fmt.Printf("- %s: %s (tools: %v)\n", a.Name, a.Description, a.Tools)
		} else {
			fmt.Printf("- %s: %s\n", a.Name, a.Description)
		}
	}

	fmt.Println("\n--- System Prompt (Progressive Disclosure Style) ---")
	fmt.Println("This is what you'd include in the system prompt:")
	fmt.Println()
	fmt.Println(p.PluginIndexSystemMessage())

	fmt.Println("\n--- Comparison: Full vs Index ---")
	fullMsg := p.ToSystemMessage()
	indexMsg := p.PluginIndexSystemMessage()
	fmt.Printf("Full system message: %d characters\n", len(fullMsg))
	fmt.Printf("Index system message: %d characters\n", len(indexMsg))
	fmt.Printf("Savings: %d characters (%.1f%% reduction)\n",
		len(fullMsg)-len(indexMsg),
		float64(len(fullMsg)-len(indexMsg))/float64(len(fullMsg))*100)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
