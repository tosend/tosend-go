// Package tosend provides a Go client for the ToSend email API.
package tosend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.tosend.com"
	defaultTimeout = 30 * time.Second
)

// Client is the ToSend API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option is a function that configures the client.
type Option func(*Client)

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithTimeout sets a custom timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// New creates a new ToSend client.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Error represents an API error response.
type Error struct {
	StatusCode int                          `json:"status_code"`
	ErrorType  string                       `json:"error_type"`
	Message    string                       `json:"message"`
	Errors     map[string]map[string]string `json:"errors,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("tosend: %s (status: %d, type: %s)", e.Message, e.StatusCode, e.ErrorType)
}

// request makes an HTTP request to the API.
func (c *Client) request(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("tosend: failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("tosend: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tosend: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("tosend: failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr Error
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return &Error{
				StatusCode: resp.StatusCode,
				ErrorType:  "unknown_error",
				Message:    string(respBody),
			}
		}
		return &apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("tosend: failed to unmarshal response: %w", err)
		}
	}

	return nil
}
