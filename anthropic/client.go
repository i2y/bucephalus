package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultBaseURL        = "https://api.anthropic.com"
	apiVersion            = "2023-06-01"
	defaultMaxTokens      = 4096
	structuredOutputsBeta = "structured-outputs-2025-11-13"
)

// client wraps the HTTP client for Anthropic API calls.
type client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// newClient creates a new Anthropic client.
func newClient(apiKey, baseURL string, httpClient *http.Client) *client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// messages sends a messages request.
func (c *client) messages(ctx context.Context, req *messagesRequest) (*messagesResponse, error) {
	// Ensure max_tokens is set
	if req.MaxTokens == 0 {
		req.MaxTokens = defaultMaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq, req.OutputFormat != nil)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, c.parseError(httpResp.StatusCode, respBody)
	}

	var resp messagesResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &resp, nil
}

// messagesStream sends a streaming messages request.
func (c *client) messagesStream(ctx context.Context, req *messagesRequest) (*streamReader, error) {
	req.Stream = true
	if req.MaxTokens == 0 {
		req.MaxTokens = defaultMaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq, req.OutputFormat != nil)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer func() { _ = httpResp.Body.Close() }()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, c.parseError(httpResp.StatusCode, respBody)
	}

	return &streamReader{
		reader: bufio.NewReader(httpResp.Body),
		closer: httpResp.Body,
	}, nil
}

func (c *client) setHeaders(req *http.Request, useStructuredOutput bool) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	if useStructuredOutput {
		req.Header.Set("anthropic-beta", structuredOutputsBeta)
	}
}

func (c *client) parseError(statusCode int, body []byte) error {
	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return &APIError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	}

	return &APIError{
		StatusCode: statusCode,
		Type:       errResp.Error.Type,
		Message:    errResp.Error.Message,
	}
}

// streamReader reads SSE events from an Anthropic stream.
type streamReader struct {
	reader *bufio.Reader
	closer io.Closer
}

// ReadEvent reads the next event from the stream.
func (s *streamReader) ReadEvent() (*streamEvent, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "event:") {
			// Read the data line
			dataLine, err := s.reader.ReadString('\n')
			if err != nil {
				return nil, err
			}

			dataLine = strings.TrimSpace(dataLine)
			if !strings.HasPrefix(dataLine, "data:") {
				continue
			}

			data := strings.TrimPrefix(dataLine, "data:")
			data = strings.TrimSpace(data)

			var event streamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				return nil, fmt.Errorf("parsing event: %w", err)
			}

			return &event, nil
		}
	}
}

// Close closes the stream.
func (s *streamReader) Close() error {
	return s.closer.Close()
}

// APIError represents an error from the Anthropic API.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("anthropic API error (status %d, type %s): %s", e.StatusCode, e.Type, e.Message)
	}
	return fmt.Sprintf("anthropic API error (status %d): %s", e.StatusCode, e.Message)
}
