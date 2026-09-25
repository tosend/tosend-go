// Package tosend provides a Go client for the ToSend email API.
package tosend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
//
// Errors carries field-level detail in either of the two shapes the API
// returns: nested from the sending endpoints ({"subject": {"required": "..."}})
// or flat from the Admin API ({"code": "insufficient_scope", "required_scope":
// "domains:read"}). Admin API errors have no ErrorType.
type Error struct {
	StatusCode int            `json:"status_code"`
	ErrorType  string         `json:"error_type,omitempty"`
	Message    string         `json:"message"`
	Errors     map[string]any `json:"errors,omitempty"`
}

func (e *Error) Error() string {
	if e.ErrorType == "" {
		return fmt.Sprintf("tosend: %s (status: %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("tosend: %s (status: %d, type: %s)", e.Message, e.StatusCode, e.ErrorType)
}

// Code returns the machine-readable code in Errors (for example
// "insufficient_scope" or "suspended_by_platform"), or "" when there is none.
func (e *Error) Code() string {
	code, _ := e.Errors["code"].(string)
	return code
}

// request makes an HTTP request to the API.
func (c *Client) request(method, path string, body interface{}, result interface{}) error {
	return c.do(method, path, nil, nil, body, result)
}

// do makes an HTTP request with optional query parameters and extra headers.
func (c *Client) do(method, path string, query url.Values, headers map[string]string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("tosend: failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, target, reqBody)
	if err != nil {
		return fmt.Errorf("tosend: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}

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
		if err := json.Unmarshal(respBody, &apiErr); err != nil || apiErr.Message == "" {
			message := strings.TrimSpace(string(respBody))
			if message == "" {
				message = http.StatusText(resp.StatusCode)
			}
			apiErr = Error{ErrorType: "unknown_error", Message: message}
		}
		if apiErr.StatusCode == 0 {
			apiErr.StatusCode = resp.StatusCode
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
