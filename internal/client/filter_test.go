// Copyright (c) Snyk Ltd.

package client

import (
	"testing"
)

func TestFilterSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "filters token auth header format",
			input:    `token abc123xyz`,
			expected: `token [FILTERED]`,
		},
		{
			name:     "filters Token auth header format (case insensitive)",
			input:    `Token ABC123XYZ`,
			expected: `Token [FILTERED]`,
		},
		{
			name:     "filters bearer token",
			input:    `bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U`,
			expected: `bearer [FILTERED]`,
		},
		{
			name:     "filters Bearer token (case insensitive)",
			input:    `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`,
			expected: `Bearer [FILTERED]`,
		},
		{
			name:     "filters client_secret in JSON",
			input:    `{"grant_type":"client_credentials","client_id":"my-client","client_secret":"super-secret-value"}`,
			expected: `{"grant_type":"client_credentials","client_id":"my-client","client_secret":"[FILTERED]"}`,
		},
		{
			name:     "filters access_token in JSON",
			input:    `{"access_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9","token_type":"bearer","expires_in":3600}`,
			expected: `{"access_token":"[FILTERED]","token_type":"bearer","expires_in":3600}`,
		},
		{
			name:     "filters api_key in JSON",
			input:    `{"api_key":"my-api-key-12345"}`,
			expected: `{"api_key":"[FILTERED]"}`,
		},
		{
			name:     "filters apikey in JSON",
			input:    `{"apikey":"another-key-value"}`,
			expected: `{"apikey":"[FILTERED]"}`,
		},
		{
			name:     "filters Authorization header",
			input:    `Authorization: token abc123xyz`,
			expected: `Authorization: token [FILTERED]`,
		},
		{
			name:     "filters Authorization header with Bearer",
			input:    `Authorization: Bearer eyJtoken123`,
			expected: `Authorization: Bearer [FILTERED]`,
		},
		{
			name:     "preserves non-sensitive data",
			input:    `{"data":{"id":"123","name":"test"},"status":"success"}`,
			expected: `{"data":{"id":"123","name":"test"},"status":"success"}`,
		},
		{
			name:     "handles multiple sensitive fields",
			input:    `{"client_secret":"secret1","access_token":"token1","api_key":"key1"}`,
			expected: `{"client_secret":"[FILTERED]","access_token":"[FILTERED]","api_key":"[FILTERED]"}`,
		},
		{
			name:     "handles empty string",
			input:    ``,
			expected: ``,
		},
		{
			name:     "filters with spaces around colon in JSON",
			input:    `{"client_secret" : "secret-with-spaces"}`,
			expected: `{"client_secret" : "[FILTERED]"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterSensitiveData(tt.input)
			if result != tt.expected {
				t.Errorf("filterSensitiveData(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
