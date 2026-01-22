// Copyright (c) Snyk Ltd.

package testutil

import (
	"bytes"
	"io"
	"net/http"
)

// NewMockResponse creates a mock HTTP response with the given status code and body
func NewMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// NewMockJSONResponse creates a mock HTTP response with JSON content type
func NewMockJSONResponse(statusCode int, body string) *http.Response {
	resp := NewMockResponse(statusCode, body)
	resp.Header.Set("Content-Type", "application/vnd.api+json")
	return resp
}

// NewMockJSONResponseWithRequestID creates a mock HTTP response with JSON content type and a snyk-request-id header
func NewMockJSONResponseWithRequestID(statusCode int, body string, requestID string) *http.Response {
	resp := NewMockJSONResponse(statusCode, body)
	resp.Header.Set("snyk-request-id", requestID)
	return resp
}
