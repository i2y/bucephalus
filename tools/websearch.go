package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/i2y/bucephalus/llm"
)

// WebSearchInput defines the input for the WebSearch tool.
type WebSearchInput struct {
	Query      string `json:"query" jsonschema:"required,description=Search query"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of results to return (default: 5)"`
}

// WebSearchOutput defines the output of the WebSearch tool.
type WebSearchOutput struct {
	Results     []SearchResult `json:"results"`
	Abstract    string         `json:"abstract,omitempty"`
	AbstractURL string         `json:"abstract_url,omitempty"`
}

// SearchResult represents a single search result.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// WebSearchTool returns the WebSearch tool.
func WebSearchTool() (llm.Tool, error) {
	return llm.NewTool(
		"web_search",
		"Search the web using DuckDuckGo. Returns search results with titles, URLs, and snippets.",
		searchWeb,
	)
}

// MustWebSearch returns the WebSearch tool, panicking on error.
func MustWebSearch() llm.Tool {
	tool, err := WebSearchTool()
	if err != nil {
		panic(err)
	}
	return tool
}

// DuckDuckGo API response structure
type ddgResponse struct {
	Abstract      string      `json:"Abstract"`
	AbstractURL   string      `json:"AbstractURL"`
	AbstractText  string      `json:"AbstractText"`
	RelatedTopics []ddgTopic  `json:"RelatedTopics"`
	Results       []ddgResult `json:"Results"`
}

type ddgTopic struct {
	Text     string `json:"Text"`
	FirstURL string `json:"FirstURL"`
	Result   string `json:"Result"`
	// Nested topics
	Topics []ddgTopic `json:"Topics"`
}

type ddgResult struct {
	Text     string `json:"Text"`
	FirstURL string `json:"FirstURL"`
	Result   string `json:"Result"`
}

func searchWeb(ctx context.Context, input WebSearchInput) (WebSearchOutput, error) {
	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}

	// Build DuckDuckGo API URL
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(input.Query))

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return WebSearchOutput{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Bucephalus/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return WebSearchOutput{}, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WebSearchOutput{}, fmt.Errorf("failed to read response: %w", err)
	}

	var ddgResp ddgResponse
	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return WebSearchOutput{}, fmt.Errorf("failed to parse response: %w", err)
	}

	var results []SearchResult

	// Add main results first
	for _, r := range ddgResp.Results {
		if len(results) >= maxResults {
			break
		}
		if r.FirstURL != "" {
			results = append(results, SearchResult{
				Title:   extractTextFromResult(r.Result),
				URL:     r.FirstURL,
				Snippet: r.Text,
			})
		}
	}

	// Add related topics
	for _, topic := range ddgResp.RelatedTopics {
		if len(results) >= maxResults {
			break
		}

		// Handle nested topics
		if len(topic.Topics) > 0 {
			for _, subtopic := range topic.Topics {
				if len(results) >= maxResults {
					break
				}
				if subtopic.FirstURL != "" {
					results = append(results, SearchResult{
						Title:   extractTextFromResult(subtopic.Result),
						URL:     subtopic.FirstURL,
						Snippet: subtopic.Text,
					})
				}
			}
		} else if topic.FirstURL != "" {
			results = append(results, SearchResult{
				Title:   extractTextFromResult(topic.Result),
				URL:     topic.FirstURL,
				Snippet: topic.Text,
			})
		}
	}

	return WebSearchOutput{
		Results:     results,
		Abstract:    ddgResp.Abstract,
		AbstractURL: ddgResp.AbstractURL,
	}, nil
}

// extractTextFromResult extracts the link text from DuckDuckGo result HTML
func extractTextFromResult(result string) string {
	// DuckDuckGo returns results with HTML like <a href="...">Title</a>...
	// We extract just the link text as the title
	if result == "" {
		return ""
	}

	// Simple extraction - find content between > and </a>
	start := 0
	for i := 0; i < len(result); i++ {
		if result[i] == '>' {
			start = i + 1
			break
		}
	}

	end := len(result)
	for i := start; i < len(result)-3; i++ {
		if result[i:i+4] == "</a>" {
			end = i
			break
		}
	}

	if start > 0 && end > start {
		return result[start:end]
	}
	return result
}
