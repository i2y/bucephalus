package gemini

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
	defaultBaseURL = "https://generativelanguage.googleapis.com"
	apiVersion     = "v1beta"
)

// client wraps the HTTP client for Gemini API calls.
type client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// newClient creates a new Gemini client.
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

// generateContent sends a generateContent request.
func (c *client) generateContent(ctx context.Context, model string, req *generateContentRequest) (*generateContentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/models/%s:generateContent", c.baseURL, apiVersion, model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

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

	var resp generateContentResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &resp, nil
}

// streamGenerateContent sends a streaming generateContent request.
func (c *client) streamGenerateContent(ctx context.Context, model string, req *generateContentRequest) (*streamReader, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/models/%s:streamGenerateContent?alt=sse", c.baseURL, apiVersion, model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

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

func (c *client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)
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
		Code:       errResp.Error.Code,
		Status:     errResp.Error.Status,
		Message:    errResp.Error.Message,
	}
}

// streamReader reads SSE events from a Gemini stream.
type streamReader struct {
	reader *bufio.Reader
	closer io.Closer
}

// ReadChunk reads the next chunk from the stream.
func (s *streamReader) ReadChunk() (*streamChunk, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Gemini streaming format: "data: {json}"
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)

			if data == "" {
				continue
			}

			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				return nil, fmt.Errorf("parsing chunk: %w", err)
			}

			return &chunk, nil
		}
	}
}

// Close closes the stream.
func (s *streamReader) Close() error {
	return s.closer.Close()
}

// APIError represents an error from the Gemini API.
type APIError struct {
	StatusCode int
	Code       int
	Status     string
	Message    string
}

func (e *APIError) Error() string {
	if e.Status != "" {
		return fmt.Sprintf("gemini API error (status %d, code %d, %s): %s", e.StatusCode, e.Code, e.Status, e.Message)
	}
	return fmt.Sprintf("gemini API error (status %d): %s", e.StatusCode, e.Message)
}
