// Copyright (c) Snyk Ltd.

package common

import (
	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
)

// ProviderData holds the configured client for use by resources and data sources
type ProviderData struct {
	Client *client.Client
}
