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

// WikipediaInput defines the input for the Wikipedia tool.
type WikipediaInput struct {
	Query    string `json:"query" jsonschema:"required,description=Search query or article title"`
	Language string `json:"language,omitempty" jsonschema:"description=Language code (default: en)"`
	Summary  bool   `json:"summary,omitempty" jsonschema:"description=Return summary only (default: true)"`
}

// WikipediaOutput defines the output of the Wikipedia tool.
type WikipediaOutput struct {
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	URL         string `json:"url"`
	Content     string `json:"content,omitempty"`
	Description string `json:"description,omitempty"`
}

// WikipediaTool returns the Wikipedia tool.
func WikipediaTool() (llm.Tool, error) {
	return llm.NewTool(
		"wikipedia",
		"Search and retrieve Wikipedia articles. Returns article summary or full content.",
		searchWikipedia,
	)
}

// MustWikipedia returns the Wikipedia tool, panicking on error.
func MustWikipedia() llm.Tool {
	tool, err := WikipediaTool()
	if err != nil {
		panic(err)
	}
	return tool
}

// Wikipedia REST API response structures
type wikiSummaryResponse struct {
	Type         string `json:"type"`
	Title        string `json:"title"`
	DisplayTitle string `json:"displaytitle"`
	Description  string `json:"description"`
	Extract      string `json:"extract"`
	ContentURLs  struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
}

type wikiSearchResponse struct {
	Pages []wikiSearchPage `json:"pages"`
}

type wikiSearchPage struct {
	ID          int    `json:"id"`
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Excerpt     string `json:"excerpt"`
}

func searchWikipedia(ctx context.Context, input WikipediaInput) (WikipediaOutput, error) {
	lang := input.Language
	if lang == "" {
		lang = "en"
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// First, search for the article
	searchURL := fmt.Sprintf("https://%s.wikipedia.org/w/rest.php/v1/search/page?q=%s&limit=1",
		lang, url.QueryEscape(input.Query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, http.NoBody)
	if err != nil {
		return WikipediaOutput{}, fmt.Errorf("failed to create search request: %w", err)
	}
	req.Header.Set("User-Agent", "Bucephalus/1.0 (https://github.com/i2y/bucephalus)")

	resp, err := client.Do(req)
	if err != nil {
		return WikipediaOutput{}, fmt.Errorf("failed to search Wikipedia: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WikipediaOutput{}, fmt.Errorf("failed to read search response: %w", err)
	}

	var searchResp wikiSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return WikipediaOutput{}, fmt.Errorf("failed to parse search response: %w", err)
	}

	if len(searchResp.Pages) == 0 {
		return WikipediaOutput{}, fmt.Errorf("no Wikipedia article found for: %s", input.Query)
	}

	articleKey := searchResp.Pages[0].Key

	// Get article summary
	summaryURL := fmt.Sprintf("https://%s.wikipedia.org/api/rest_v1/page/summary/%s",
		lang, url.PathEscape(articleKey))

	req, err = http.NewRequestWithContext(ctx, "GET", summaryURL, http.NoBody)
	if err != nil {
		return WikipediaOutput{}, fmt.Errorf("failed to create summary request: %w", err)
	}
	req.Header.Set("User-Agent", "Bucephalus/1.0 (https://github.com/i2y/bucephalus)")

	resp, err = client.Do(req)
	if err != nil {
		return WikipediaOutput{}, fmt.Errorf("failed to fetch Wikipedia summary: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return WikipediaOutput{}, fmt.Errorf("failed to read summary response: %w", err)
	}

	var summaryResp wikiSummaryResponse
	if err := json.Unmarshal(body, &summaryResp); err != nil {
		return WikipediaOutput{}, fmt.Errorf("failed to parse summary response: %w", err)
	}

	output := WikipediaOutput{
		Title:       summaryResp.Title,
		Summary:     summaryResp.Extract,
		URL:         summaryResp.ContentURLs.Desktop.Page,
		Description: summaryResp.Description,
	}

	// If full content requested, fetch it
	if !input.Summary {
		content, err := fetchWikipediaContent(ctx, client, lang, articleKey)
		if err == nil {
			output.Content = content
		}
	}

	return output, nil
}

func fetchWikipediaContent(ctx context.Context, client *http.Client, lang, articleKey string) (string, error) {
	// Use the mobile-html endpoint for cleaner content
	contentURL := fmt.Sprintf("https://%s.wikipedia.org/api/rest_v1/page/mobile-html/%s",
		lang, url.PathEscape(articleKey))

	req, err := http.NewRequestWithContext(ctx, "GET", contentURL, http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Bucephalus/1.0 (https://github.com/i2y/bucephalus)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	// Limit to 500KB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}

	// Convert HTML to text
	return htmlToText(string(body)), nil
}
