package tools

import "github.com/i2y/bucephalus/llm"

// AllTools returns all built-in tools.
func AllTools() []llm.Tool {
	return []llm.Tool{
		MustRead(),
		MustWrite(),
		MustGlob(),
		MustGrep(),
		MustBash(),
		MustWebFetch(),
		MustWebSearch(),
		MustWikipedia(),
	}
}

// FileTools returns file-related tools only.
// Includes: Read, Write, Glob, Grep
func FileTools() []llm.Tool {
	return []llm.Tool{
		MustRead(),
		MustWrite(),
		MustGlob(),
		MustGrep(),
	}
}

// WebTools returns web-related tools only.
// Includes: WebFetch, WebSearch, Wikipedia
func WebTools() []llm.Tool {
	return []llm.Tool{
		MustWebFetch(),
		MustWebSearch(),
		MustWikipedia(),
	}
}

// KnowledgeTools returns knowledge retrieval tools.
// Includes: WebSearch, Wikipedia
func KnowledgeTools() []llm.Tool {
	return []llm.Tool{
		MustWebSearch(),
		MustWikipedia(),
	}
}

// ReadOnlyTools returns tools that don't modify the filesystem.
// Includes: Read, Glob, Grep, WebFetch, WebSearch, Wikipedia
func ReadOnlyTools() []llm.Tool {
	return []llm.Tool{
		MustRead(),
		MustGlob(),
		MustGrep(),
		MustWebFetch(),
		MustWebSearch(),
		MustWikipedia(),
	}
}

// SystemTools returns tools that can modify the system.
// Includes: Write, Bash
func SystemTools() []llm.Tool {
	return []llm.Tool{
		MustWrite(),
		MustBash(),
	}
}
