// Copyright (c) Snyk Ltd.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// HTTPDoer is an interface for making HTTP requests
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Authenticator is an interface for adding authentication to requests
type Authenticator interface {
	Authenticate(req *http.Request) error
}

// Client is the Snyk API client
type Client struct {
	baseURL    string
	httpClient HTTPDoer
	auth       Authenticator
	apiVersion string
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient HTTPDoer) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithAPIVersion sets the API version to use
func WithAPIVersion(version string) ClientOption {
	return func(c *Client) {
		c.apiVersion = version
	}
}

// NewClient creates a new Snyk API client
func NewClient(baseURL string, auth Authenticator, opts ...ClientOption) *Client {
	c := &Client{
		baseURL:    baseURL,
		auth:       auth,
		httpClient: &http.Client{
			// No timeout here - rely on context timeout from Terraform resource timeouts
			// This allows operations like delete to respect the user-configured timeouts
		},
		apiVersion: "2024-10-15",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// RegionToBaseURL returns the API base URL for a given region
func RegionToBaseURL(region string) string {
	switch region {
	case "eu":
		return "https://api.eu.snyk.io"
	case "au":
		return "https://api.au.snyk.io"
	case "us", "":
		return "https://api.snyk.io"
	default:
		return "https://api.snyk.io"
	}
}

// APIError represents an error response from the Snyk API
type APIError struct {
	StatusCode int
	Message    string
	Body       string
	RequestID  string
}

func (e *APIError) Error() string {
	var msg string
	if e.Body != "" {
		msg = fmt.Sprintf("API error (status %d): %s - %s", e.StatusCode, e.Message, e.Body)
	} else {
		msg = fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	if e.RequestID != "" {
		msg = fmt.Sprintf("%s (snyk-request-id: %s)", msg, e.RequestID)
	}
	return msg
}

// sensitivePatterns defines regex patterns for sensitive data that should be filtered from logs
var sensitivePatterns = []*regexp.Regexp{
	// API tokens in Authorization header format
	regexp.MustCompile(`(?i)(token\s+)[a-zA-Z0-9_-]+`),
	// Bearer tokens
	regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9._-]+`),
	// client_secret in JSON
	regexp.MustCompile(`(?i)("client_secret"\s*:\s*")[^"]+`),
	// access_token in JSON
	regexp.MustCompile(`(?i)("access_token"\s*:\s*")[^"]+`),
	// Generic API key patterns
	regexp.MustCompile(`(?i)("api_key"\s*:\s*")[^"]+`),
	regexp.MustCompile(`(?i)("apikey"\s*:\s*")[^"]+`),
	// Authorization header value
	regexp.MustCompile(`(?i)(authorization:\s*)(token|bearer)\s+[a-zA-Z0-9._-]+`),
}

// filterSensitiveData replaces sensitive tokens and secrets with [FILTERED] in the given string
func filterSensitiveData(data string) string {
	result := data
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Find the prefix part and replace only the sensitive value
			loc := pattern.FindStringSubmatchIndex(match)
			if len(loc) >= 4 {
				prefix := match[:loc[3]]
				return prefix + "[FILTERED]"
			}
			return "[FILTERED]"
		})
	}
	return result
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	var reqBodyBytes []byte
	if body != nil {
		var err error
		reqBodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(reqBodyBytes)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")

	if err := c.auth.Authenticate(req); err != nil {
		return nil, fmt.Errorf("failed to authenticate request: %w", err)
	}

	// Log the request details at trace level
	logFields := map[string]interface{}{
		"method": method,
		"url":    url,
	}
	if len(reqBodyBytes) > 0 {
		logFields["request_body"] = filterSensitiveData(string(reqBodyBytes))
	}
	tflog.Trace(ctx, "Snyk API request", logFields)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// handleResponse processes an API response and unmarshals it into the target
func (c *Client) handleResponse(ctx context.Context, resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the response details at trace level
	logFields := map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
	}
	if len(bodyBytes) > 0 {
		logFields["response_body"] = filterSensitiveData(string(bodyBytes))
	}
	tflog.Trace(ctx, "Snyk API response", logFields)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
			Body:       string(bodyBytes),
			RequestID:  resp.Header.Get("snyk-request-id"),
		}
	}

	if target != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, target); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, target interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, target)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}, target interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, target)
}

// Patch performs a PATCH request
func (c *Client) Patch(ctx context.Context, path string, body interface{}, target interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, target)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, nil)
}

// GetAPIVersion returns the API version being used
func (c *Client) GetAPIVersion() string {
	return c.apiVersion
}
