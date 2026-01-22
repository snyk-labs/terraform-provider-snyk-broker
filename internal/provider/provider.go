// Copyright (c) Snyk Ltd.

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
	"github.com/snyk/terraform-provider-snyk-broker/internal/common"
	"github.com/snyk/terraform-provider-snyk-broker/internal/datasources"
	"github.com/snyk/terraform-provider-snyk-broker/internal/resources"
)

// Ensure SnykBrokerProvider satisfies various provider interfaces.
var _ provider.Provider = &SnykBrokerProvider{}

// SnykBrokerProvider defines the provider implementation.
type SnykBrokerProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// SnykBrokerProviderModel describes the provider data model.
type SnykBrokerProviderModel struct {
	APIToken     types.String `tfsdk:"api_token"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Region       types.String `tfsdk:"region"`
}

func (p *SnykBrokerProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "snyk"
	resp.Version = p.version
}

func (p *SnykBrokerProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Snyk Broker provider allows you to manage Snyk Broker resources including deployments, connections, credentials, and integrations.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Description: "The Snyk API token. Can also be set via the SNYK_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"client_id": schema.StringAttribute{
				Description: "The OAuth client ID for service account authentication. Can also be set via the SNYK_CLIENT_ID environment variable.",
				Optional:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The OAuth client secret for service account authentication. Can also be set via the SNYK_CLIENT_SECRET environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"region": schema.StringAttribute{
				Description: "The Snyk region to use (us, eu, au). Defaults to us. Can also be set via the SNYK_REGION environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *SnykBrokerProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Snyk Broker provider")

	var config SnykBrokerProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get values from config or environment
	apiToken := getConfigOrEnv(config.APIToken, "SNYK_TOKEN")
	clientID := getConfigOrEnv(config.ClientID, "SNYK_CLIENT_ID")
	clientSecret := getConfigOrEnv(config.ClientSecret, "SNYK_CLIENT_SECRET")
	region := getConfigOrEnv(config.Region, "SNYK_REGION")

	if region == "" {
		region = "us"
	}

	// Validate authentication configuration
	hasToken := apiToken != ""
	hasOAuth := clientID != "" && clientSecret != ""

	if !hasToken && !hasOAuth {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Authentication Configuration",
			"Either api_token or both client_id and client_secret must be provided. "+
				"You can also set these via SNYK_TOKEN or SNYK_CLIENT_ID/SNYK_CLIENT_SECRET environment variables.",
		)
		return
	}

	// Create authenticator based on configuration
	var auth client.Authenticator
	if hasToken {
		auth = client.NewTokenAuthenticator(apiToken)
	} else {
		tokenURL := client.OAuthTokenURL(region)
		auth = client.NewOAuthAuthenticator(clientID, clientSecret, tokenURL)
	}

	// Create the API client
	baseURL := client.RegionToBaseURL(region)
	apiClient := client.NewClient(baseURL, auth)

	tflog.Debug(ctx, "Snyk Broker provider configured", map[string]interface{}{
		"region":   region,
		"base_url": baseURL,
		"auth":     map[bool]string{true: "token", false: "oauth"}[hasToken],
	})

	// Make the client available to resources and data sources
	providerData := &common.ProviderData{
		Client: apiClient,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *SnykBrokerProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewBrokerAppInstallResource,
		resources.NewBrokerDeploymentResource,
		resources.NewBrokerCredentialResource,
		resources.NewBrokerConnectionResource,
		resources.NewBrokerConnectionIntegrationResource,
		resources.NewBrokerBulkMigrationResource,
	}
}

func (p *SnykBrokerProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewBrokerDeploymentsDataSource,
		datasources.NewBrokerConnectionsDataSource,
		datasources.NewBrokerConnectionsForOrgDataSource,
		datasources.NewBrokerConnectionIntegrationsDataSource,
		datasources.NewBrokerMigrationOrgsDataSource,
	}
}

// New creates a new provider instance
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SnykBrokerProvider{
			version: version,
		}
	}
}

// getConfigOrEnv returns the config value if set, otherwise the environment variable value
func getConfigOrEnv(configValue types.String, envVar string) string {
	if !configValue.IsNull() && !configValue.IsUnknown() {
		return configValue.ValueString()
	}
	return os.Getenv(envVar)
}
