// Copyright (c) Snyk Ltd.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// TokenAuthenticator authenticates requests using an API token
type TokenAuthenticator struct {
	token string
}

// NewTokenAuthenticator creates a new token-based authenticator
func NewTokenAuthenticator(token string) *TokenAuthenticator {
	return &TokenAuthenticator{token: token}
}

// Authenticate adds the API token to the request
func (a *TokenAuthenticator) Authenticate(req *http.Request) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", a.token))
	return nil
}

// OAuthAuthenticator authenticates requests using OAuth client credentials
type OAuthAuthenticator struct {
	clientID     string
	clientSecret string
	tokenURL     string
	httpClient   HTTPDoer

	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time
}

// OAuthTokenResponse represents the OAuth token response
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// NewOAuthAuthenticator creates a new OAuth-based authenticator
func NewOAuthAuthenticator(clientID, clientSecret, tokenURL string) *OAuthAuthenticator {
	return &OAuthAuthenticator{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithHTTPClient sets a custom HTTP client for the OAuth authenticator
func (a *OAuthAuthenticator) WithHTTPClient(client HTTPDoer) *OAuthAuthenticator {
	a.httpClient = client
	return a
}

// Authenticate adds the OAuth access token to the request
func (a *OAuthAuthenticator) Authenticate(req *http.Request) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	token, err := a.getValidToken(req.Context())
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return nil
}

// getValidToken returns a valid access token, refreshing if necessary
func (a *OAuthAuthenticator) getValidToken(ctx context.Context) (string, error) {
	a.mu.RLock()
	if a.accessToken != "" && time.Now().Before(a.expiresAt.Add(-1*time.Minute)) {
		token := a.accessToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	return a.refreshToken(ctx)
}

// refreshToken obtains a new access token
func (a *OAuthAuthenticator) refreshToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring lock
	if a.accessToken != "" && time.Now().Before(a.expiresAt.Add(-1*time.Minute)) {
		return a.accessToken, nil
	}

	body := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     a.clientID,
		"client_secret": a.clientSecret,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token request: %w", err)
	}

	// Log the OAuth token request at trace level (with sensitive data filtered)
	tflog.Trace(ctx, "Snyk OAuth token request", map[string]interface{}{
		"method": http.MethodPost,
		"url":    a.tokenURL,
		// Don't log the body as it contains client_secret
		"request_body": filterSensitiveData(string(jsonBody)),
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.tokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute token request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response body: %w", err)
	}

	// Log the OAuth token response at trace level (with sensitive data filtered)
	tflog.Trace(ctx, "Snyk OAuth token response", map[string]interface{}{
		"status_code":   resp.StatusCode,
		"status":        resp.Status,
		"response_body": filterSensitiveData(string(bodyBytes)),
	})

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	a.accessToken = tokenResp.AccessToken
	a.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return a.accessToken, nil
}

// OAuthTokenURL returns the OAuth token URL for a given region
func OAuthTokenURL(region string) string {
	switch region {
	case "eu":
		return "https://api.eu.snyk.io/oauth2/token"
	case "au":
		return "https://api.au.snyk.io/oauth2/token"
	case "us", "":
		return "https://api.snyk.io/oauth2/token"
	default:
		return "https://api.snyk.io/oauth2/token"
	}
}
