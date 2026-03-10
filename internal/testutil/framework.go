// Copyright (c) Snyk Ltd.

package testutil

import (
	"github.com/snyk-labs/snyk-broker-provider/internal/client"
	"github.com/snyk-labs/snyk-broker-provider/internal/common"
)

// CreateMockProviderData creates a mock ProviderData with a configured client
func CreateMockProviderData(mockHTTP *MockHTTPClient) *common.ProviderData {
	mockAuth := &MockAuthenticator{}
	c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))
	return &common.ProviderData{
		Client: c,
	}
}
