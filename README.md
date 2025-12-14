# Bucephalus

A Go LLM client library that provides a unified API across multiple providers (OpenAI, Anthropic, Gemini).

## Installation

```bash
go get github.com/i2y/bucephalus
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    _ "github.com/i2y/bucephalus/anthropic"
    "github.com/i2y/bucephalus/llm"
)

func main() {
    ctx := context.Background()

    resp, err := llm.Call(ctx, "Hello, world!",
        llm.WithProvider("anthropic"),
        llm.WithModel("claude-3-5-haiku-latest"),
    )
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Text())
}
```

## Features

### Basic Calls

```go
// Simple call
resp, _ := llm.Call(ctx, "What is Go?",
    llm.WithProvider("openai"),
    llm.WithModel("gpt-4o"),
)
fmt.Println(resp.Text())

// Call with message array
messages := []llm.Message{
    llm.SystemMessage("You are a helpful assistant."),
    llm.UserMessage("Hello!"),
}
resp, _ := llm.CallMessages(ctx, messages, opts...)
```

### Structured Output

```go
type Recipe struct {
    Name        string   `json:"name"`
    Ingredients []string `json:"ingredients"`
}

resp, _ := llm.CallParse[Recipe](ctx, "Give me a pasta recipe", opts...)
recipe, _ := resp.Parsed()
fmt.Println(recipe.Name)
```

### Streaming

```go
stream, _ := llm.CallStream(ctx, "Tell me a story", opts...)
for chunk, err := range stream {
    if err != nil {
        break
    }
    fmt.Print(chunk.Text())
}
```

### Multi-turn Conversations (Resume)

```go
resp1, _ := llm.Call(ctx, "Recommend a book", opts...)
fmt.Println(resp1.Text())

// Continue the conversation
resp2, _ := resp1.Resume(ctx, "Why did you recommend that?")
fmt.Println(resp2.Text())
```

### Tool Calling

```go
weatherTool := llm.NewTool(
    "get_weather",
    "Get weather for a city",
    func(ctx context.Context, args struct{ City string }) (string, error) {
        return fmt.Sprintf("Weather in %s: Sunny, 22°C", args.City), nil
    },
)

resp, _ := llm.Call(ctx, "What's the weather in Tokyo?",
    llm.WithTools(weatherTool),
    opts...,
)

if resp.HasToolCalls() {
    results, _ := llm.ExecuteToolCalls(ctx, resp.ToolCalls(), registry)
    resp2, _ := resp.ResumeWithToolOutputs(ctx, results)
    fmt.Println(resp2.Text())
}
```

### Built-in Tools

The `tools` package provides ready-to-use tools for common operations.

```go
import "github.com/i2y/bucephalus/tools"

// Use all built-in tools
resp, _ := llm.Call(ctx, "Find all Go files with 'TODO' comments",
    llm.WithProvider("anthropic"),
    llm.WithModel("claude-3-5-haiku-latest"),
    llm.WithTools(tools.AllTools()...),
)

// Use specific tool groups
resp, _ := llm.Call(ctx, "Read the README.md file",
    llm.WithTools(tools.FileTools()...),
)

// Search Wikipedia
resp, _ := llm.Call(ctx, "What is Go programming language?",
    llm.WithTools(tools.KnowledgeTools()...),
)
```

**Available Tools:**

| Tool | Description |
|------|-------------|
| `Read` | Read file contents with offset/limit support |
| `Write` | Write content to a file (creates directories) |
| `Glob` | Find files matching a glob pattern (`**/*.go`) |
| `Grep` | Search files with regular expressions |
| `Bash` | Execute shell commands with timeout |
| `WebFetch` | Fetch and extract content from URLs |
| `WebSearch` | Search the web (DuckDuckGo) |
| `Wikipedia` | Search and retrieve Wikipedia articles |

**Tool Groups:**

| Function | Tools |
|----------|-------|
| `AllTools()` | All 8 tools |
| `FileTools()` | Read, Write, Glob, Grep |
| `WebTools()` | WebFetch, WebSearch, Wikipedia |
| `KnowledgeTools()` | WebSearch, Wikipedia |
| `ReadOnlyTools()` | Read, Glob, Grep, WebFetch, WebSearch, Wikipedia |
| `SystemTools()` | Write, Bash |

### Plugin Support (Claude Code-style)

Load plugins using a directory structure similar to Claude Code.

```go
import "github.com/i2y/bucephalus/plugin"

p, _ := plugin.Load("./my-plugin")

// Command expansion with $ARGUMENTS substitution
expanded, _ := p.ExpandCommand("/translate Hello World")
resp, _ := llm.Call(ctx, expanded.Arguments, expanded.ToOption())

// Auto-detect slash commands
opt, userMsg, _ := p.ProcessInput("/greet John")
resp, _ := llm.Call(ctx, userMsg, opt, otherOpts...)

// Use skill context
skill := p.GetSkill("code-review")
resp, _ := llm.Call(ctx, "Review this code", skill.ToOption())

// Run agent with conversation context
agent := p.GetAgent("helper")
runner := agent.NewRunner(
    plugin.WithAgentProvider("anthropic"),
    plugin.WithAgentModel("claude-3-5-haiku-latest"),
    plugin.WithAgentTools(tools...),  // Filtered by agent.Tools
    // Pass additional llm.Options for all Run() calls
    plugin.WithAgentLLMOptions(
        llm.WithTopP(0.9),
        llm.WithSystemMessage(p.SkillsIndexSystemMessage()),
    ),
)

// Multi-turn conversation - context is maintained across calls
resp1, _ := runner.Run(ctx, "My name is Alice")
resp2, _ := runner.Run(ctx, "What is my name?")  // Remembers "Alice"

// Pass options for a specific Run() call only
resp3, _ := runner.Run(ctx, "Help me with this task",
    plugin.WithRunSystemMessage("Extra context for this call"),
    plugin.WithRunLLMOptions(llm.WithTemperature(0.5)),
)

// Access conversation history and state
history := runner.Context().History()
runner.Context().SetState("user_id", 123)
runner.ClearHistory()  // Clear conversation, keep state

// Progressive Disclosure (Claude Code style)
// Include only metadata in system prompt, load full content when needed
indexMsg := p.PluginIndexSystemMessage()  // ~60% smaller than full content
resp, _ := llm.Call(ctx, "Help me with code quality",
    llm.WithSystemMessage(indexMsg),  // Compact skills/commands list
)
```

**Supported structure:**
- `.claude-plugin/plugin.json` - Manifest
- `commands/*.md` - Slash commands (with `$ARGUMENTS` substitution)
- `agents/*.md` - Sub-agents (with conversation context)
- `skills/*/SKILL.md` - Skills

## Options

### LLM Call Options

| Option | Description |
|--------|-------------|
| `WithProvider(name)` | Select provider ("openai", "anthropic", "gemini") |
| `WithModel(name)` | Select model |
| `WithTemperature(t)` | Sampling temperature (0.0-2.0) |
| `WithMaxTokens(n)` | Maximum tokens |
| `WithTopP(p)` | Nucleus sampling |
| `WithTopK(k)` | Top-K (Anthropic, Gemini) |
| `WithSeed(s)` | Seed value (OpenAI only) |
| `WithStopSequences(...)` | Stop sequences |
| `WithSystemMessage(msg)` | System message |
| `WithTools(...)` | Tool definitions |

### AgentRunner Options

| Option | Description |
|--------|-------------|
| `WithAgentProvider(name)` | Set provider for the agent |
| `WithAgentModel(name)` | Set model for the agent |
| `WithAgentTools(...)` | Provide tools (filtered by agent.Tools) |
| `WithAgentTemperature(t)` | Set temperature |
| `WithAgentMaxTokens(n)` | Set max tokens |
| `WithAgentContext(ctx)` | Share context between agents |
| `WithAgentLLMOptions(...)` | Pass additional llm.Options for all Run() calls |

### Run Options (per-call)

| Option | Description |
|--------|-------------|
| `WithRunSystemMessage(msg)` | Add extra system message for this call only |
| `WithRunLLMOptions(...)` | Add extra llm.Options for this call only |

## Package Structure

```
llm/          # Public API
provider/     # Provider interface
openai/       # OpenAI implementation
anthropic/    # Anthropic implementation
gemini/       # Google Gemini implementation
schema/       # JSON schema generation
mcp/          # Model Context Protocol integration
plugin/       # Claude Code Plugin loader
tools/        # Built-in tools (Read, Write, Glob, Grep, Bash, Web)
```

## License

MIT
