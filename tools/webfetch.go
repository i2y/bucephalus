package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/i2y/bucephalus/llm"
)

// WebFetchInput defines the input for the WebFetch tool.
type WebFetchInput struct {
	URL     string `json:"url" jsonschema:"required,description=URL to fetch"`
	Extract string `json:"extract,omitempty" jsonschema:"description=Extract mode: html (raw), text (stripped), or markdown (default: text)"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"description=Timeout in seconds (default: 30)"`
}

// WebFetchOutput defines the output of the WebFetch tool.
type WebFetchOutput struct {
	Content    string `json:"content"`
	StatusCode int    `json:"status_code"`
	Title      string `json:"title,omitempty"`
	URL        string `json:"url"`
}

// WebFetchTool returns the WebFetch tool.
func WebFetchTool() (llm.Tool, error) {
	return llm.NewTool(
		"web_fetch",
		"Fetch content from a URL. Returns the page content with optional extraction mode.",
		fetchURL,
	)
}

// MustWebFetch returns the WebFetch tool, panicking on error.
func MustWebFetch() llm.Tool {
	tool, err := WebFetchTool()
	if err != nil {
		panic(err)
	}
	return tool
}

func fetchURL(ctx context.Context, input WebFetchInput) (WebFetchOutput, error) {
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", input.URL, http.NoBody)
	if err != nil {
		return WebFetchOutput{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Bucephalus/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return WebFetchOutput{}, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Limit response size to 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return WebFetchOutput{}, fmt.Errorf("failed to read response: %w", err)
	}

	content := string(body)
	title := extractTitle(content)

	// Apply extraction mode
	extract := input.Extract
	if extract == "" {
		extract = "text"
	}

	switch extract {
	case "html":
		// Return raw HTML
	case "text":
		content = htmlToText(content)
	case "markdown":
		content = htmlToMarkdown(content)
	}

	return WebFetchOutput{
		Content:    content,
		StatusCode: resp.StatusCode,
		Title:      title,
		URL:        resp.Request.URL.String(),
	}, nil
}

func extractTitle(html string) string {
	re := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func htmlToText(html string) string {
	// Remove script and style elements (separate patterns since Go regex doesn't support backreferences)
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = styleRe.ReplaceAllString(html, "")

	// Remove HTML comments
	commentRe := regexp.MustCompile(`<!--.*?-->`)
	html = commentRe.ReplaceAllString(html, "")

	// Replace common block elements with newlines
	blockRe := regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|tr|br)[^>]*>`)
	html = blockRe.ReplaceAllString(html, "\n")

	// Remove all remaining HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	text := tagRe.ReplaceAllString(html, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Normalize whitespace
	spaceRe := regexp.MustCompile(`[ \t]+`)
	text = spaceRe.ReplaceAllString(text, " ")

	// Normalize newlines
	newlineRe := regexp.MustCompile(`\n{3,}`)
	text = newlineRe.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

func htmlToMarkdown(html string) string {
	// Start with text extraction
	result := html

	// Remove script and style (separate patterns since Go regex doesn't support backreferences)
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	result = scriptRe.ReplaceAllString(result, "")
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	result = styleRe.ReplaceAllString(result, "")

	// Convert headers
	for i := 1; i <= 6; i++ {
		prefix := strings.Repeat("#", i)
		headerRe := regexp.MustCompile(fmt.Sprintf(`(?is)<h%d[^>]*>(.*?)</h%d>`, i, i))
		result = headerRe.ReplaceAllString(result, prefix+" $1\n\n")
	}

	// Convert links
	linkRe := regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	result = linkRe.ReplaceAllString(result, "[$2]($1)")

	// Convert bold (separate patterns since Go regex doesn't support backreferences)
	strongRe := regexp.MustCompile(`(?is)<strong[^>]*>(.*?)</strong>`)
	result = strongRe.ReplaceAllString(result, "**$1**")
	bRe := regexp.MustCompile(`(?is)<b[^>]*>(.*?)</b>`)
	result = bRe.ReplaceAllString(result, "**$1**")

	// Convert italic (separate patterns since Go regex doesn't support backreferences)
	emRe := regexp.MustCompile(`(?is)<em[^>]*>(.*?)</em>`)
	result = emRe.ReplaceAllString(result, "*$1*")
	iRe := regexp.MustCompile(`(?is)<i[^>]*>(.*?)</i>`)
	result = iRe.ReplaceAllString(result, "*$1*")

	// Convert code
	codeRe := regexp.MustCompile(`(?is)<code[^>]*>(.*?)</code>`)
	result = codeRe.ReplaceAllString(result, "`$1`")

	// Convert lists
	liRe := regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	result = liRe.ReplaceAllString(result, "- $1\n")

	// Convert paragraphs
	pRe := regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	result = pRe.ReplaceAllString(result, "$1\n\n")

	// Convert br
	brRe := regexp.MustCompile(`(?i)<br[^>]*>`)
	result = brRe.ReplaceAllString(result, "\n")

	// Remove remaining tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	result = tagRe.ReplaceAllString(result, "")

	// Decode entities
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#39;", "'")

	// Clean up whitespace
	spaceRe := regexp.MustCompile(`[ \t]+`)
	result = spaceRe.ReplaceAllString(result, " ")
	newlineRe := regexp.MustCompile(`\n{3,}`)
	result = newlineRe.ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}
