package openai

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

const defaultBaseURL = "https://api.openai.com/v1"

// client wraps the HTTP client for OpenAI API calls.
type client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// newClient creates a new OpenAI client.
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

// chatCompletion sends a chat completion request.
func (c *client) chatCompletion(ctx context.Context, req *chatCompletionRequest) (*chatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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

	var resp chatCompletionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &resp, nil
}

// parseError parses an error response from the API.
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
		Message:    errResp.Error.Message,
		Type:       errResp.Error.Type,
		Code:       errResp.Error.Code,
	}
}

// APIError represents an error from the OpenAI API.
type APIError struct {
	StatusCode int
	Message    string
	Type       string
	Code       string
}

func (e *APIError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("openai API error (status %d, type %s): %s", e.StatusCode, e.Type, e.Message)
	}
	return fmt.Sprintf("openai API error (status %d): %s", e.StatusCode, e.Message)
}

// chatCompletionStream sends a streaming chat completion request.
func (c *client) chatCompletionStream(ctx context.Context, req *chatCompletionRequest) (*streamReader, error) {
	// Create a copy with stream enabled
	streamReq := *req
	streamReq.Stream = true

	body, err := json.Marshal(streamReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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

// streamReader reads SSE events from an OpenAI stream.
type streamReader struct {
	reader *bufio.Reader
	closer io.Closer
}

// ReadChunk reads the next chunk from the stream.
// Returns nil, io.EOF when the stream is done.
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

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)

		if data == "[DONE]" {
			return nil, io.EOF
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("parsing chunk: %w", err)
		}

		return &chunk, nil
	}
}

// Close closes the stream.
func (s *streamReader) Close() error {
	return s.closer.Close()
}
