package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// HTTPClient provides utilities for making HTTP requests in tests
type HTTPClient struct {
	BaseURL string
	Headers map[string]string
	t       *testing.T
}

// NewHTTPClient creates a new HTTP client for testing
func NewHTTPClient(t *testing.T, baseURL string) *HTTPClient {
	return &HTTPClient{
		BaseURL: baseURL,
		Headers: make(map[string]string),
		t:       t,
	}
}

// WithHeaders adds headers to the HTTP client
func (c *HTTPClient) WithHeaders(headers map[string]string) *HTTPClient {
	for k, v := range headers {
		c.Headers[k] = v
	}
	return c
}

// GET makes a GET request to the specified path
func (c *HTTPClient) GET(path string) *HTTPResponse {
	return c.Request("GET", path, nil)
}

// POST makes a POST request to the specified path with JSON body
func (c *HTTPClient) POST(path string, body interface{}) *HTTPResponse {
	return c.Request("POST", path, body)
}

// PUT makes a PUT request to the specified path with JSON body
func (c *HTTPClient) PUT(path string, body interface{}) *HTTPResponse {
	return c.Request("PUT", path, body)
}

// DELETE makes a DELETE request to the specified path
func (c *HTTPClient) DELETE(path string) *HTTPResponse {
	return c.Request("DELETE", path, nil)
}

// Request makes an HTTP request with the specified method, path, and body
func (c *HTTPClient) Request(method, path string, body interface{}) *HTTPResponse {
	url := c.BaseURL + path

	var requestBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(c.t, err, "Failed to marshal request body")
		requestBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, requestBody)
	require.NoError(c.t, err, fmt.Sprintf("Failed to create %s request to %s", method, url))

	// Add headers
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	// If we have a JSON body, set content type
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(c.t, err, fmt.Sprintf("Failed to execute %s request to %s", method, url))

	return &HTTPResponse{
		Response: resp,
		t:        c.t,
	}
}

// HTTPResponse wraps http.Response with testing utilities
type HTTPResponse struct {
	*http.Response
	t *testing.T
}

// ExpectStatus asserts that the response has the expected status code
func (r *HTTPResponse) ExpectStatus(expected int) *HTTPResponse {
	require.Equal(r.t, expected, r.StatusCode,
		fmt.Sprintf("Expected status %d, got %d. Response: %s", expected, r.StatusCode, r.BodyString()))
	return r
}

// ExpectJSON decodes the response body as JSON into the provided struct
func (r *HTTPResponse) ExpectJSON(target interface{}) *HTTPResponse {
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	require.NoError(r.t, err, "Failed to read response body")

	err = json.Unmarshal(body, target)
	require.NoError(r.t, err, fmt.Sprintf("Failed to unmarshal JSON response: %s", string(body)))

	return r
}

// BodyString returns the response body as a string
func (r *HTTPResponse) BodyString() string {
	if r.Body == nil {
		return ""
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Sprintf("Error reading body: %v", err)
	}

	// Reset body for potential future reads
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return string(body)
}

// ExpectHeader asserts that the response has the expected header value
func (r *HTTPResponse) ExpectHeader(key, expected string) *HTTPResponse {
	actual := r.Header.Get(key)
	require.Equal(r.t, expected, actual,
		fmt.Sprintf("Expected header %s to be '%s', got '%s'", key, expected, actual))
	return r
}

// ExpectContains asserts that the response body contains the expected string
func (r *HTTPResponse) ExpectContains(expected string) *HTTPResponse {
	body := r.BodyString()
	require.Contains(r.t, body, expected,
		fmt.Sprintf("Expected response body to contain '%s', got: %s", expected, body))
	return r
}

// Close closes the response body
func (r *HTTPResponse) Close() {
	if r.Body != nil {
		r.Body.Close()
	}
}
