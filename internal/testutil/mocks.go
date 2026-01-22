// Copyright (c) Snyk Ltd.

package testutil

import (
	"net/http"
	"net/http/httptest"
)

// MockHTTPClient implements client.HTTPDoer for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do executes the mock function
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// NewTestServer creates an httptest.Server with predefined responses
func NewTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// MockAuthenticator implements client.Authenticator for testing
type MockAuthenticator struct {
	AuthenticateFunc func(req *http.Request) error
}

// Authenticate executes the mock function
func (m *MockAuthenticator) Authenticate(req *http.Request) error {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(req)
	}
	req.Header.Set("Authorization", "token mock-token")
	return nil
}
