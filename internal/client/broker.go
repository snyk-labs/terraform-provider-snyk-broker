// Copyright (c) Snyk Ltd.

package client

import (
	"context"
	"fmt"
)

// BrokerAppInstallRequest represents a request to install the Broker app
type BrokerAppInstallRequest struct {
	Data          BrokerAppInstallData          `json:"data"`
	Relationships BrokerAppInstallRelationships `json:"relationships"`
}

type BrokerAppInstallData struct {
	Type string `json:"type"`
}

type BrokerAppInstallRelationships struct {
	App BrokerAppInstallAppRelationship `json:"app"`
}

type BrokerAppInstallAppRelationship struct {
	Data BrokerAppInstallAppData `json:"data"`
}

type BrokerAppInstallAppData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// BrokerAppInstallResponse represents the response from installing the Broker app
type BrokerAppInstallResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			ClientID string `json:"client_id"`
		} `json:"attributes"`
	} `json:"data"`
}

// InstallBrokerApp installs the Snyk Broker app to an organization
func (c *Client) InstallBrokerApp(ctx context.Context, orgID, appID string) (*BrokerAppInstallResponse, error) {
	path := fmt.Sprintf("/rest/orgs/%s/apps/installs?version=%s", orgID, c.apiVersion)

	req := BrokerAppInstallRequest{
		Data: BrokerAppInstallData{
			Type: "app_install",
		},
		Relationships: BrokerAppInstallRelationships{
			App: BrokerAppInstallAppRelationship{
				Data: BrokerAppInstallAppData{
					ID:   appID,
					Type: "app",
				},
			},
		},
	}

	var resp BrokerAppInstallResponse
	if err := c.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to install broker app: %w", err)
	}

	return &resp, nil
}

// UninstallBrokerApp uninstalls the Snyk Broker app from an organization
func (c *Client) UninstallBrokerApp(ctx context.Context, orgID, installID string) error {
	path := fmt.Sprintf("/rest/orgs/%s/apps/installs/%s?version=%s", orgID, installID, c.apiVersion)

	if err := c.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to uninstall broker app: %w", err)
	}

	return nil
}

// BrokerDeployment represents a broker deployment
type BrokerDeployment struct {
	ID         string                     `json:"id"`
	Type       string                     `json:"type"`
	Attributes BrokerDeploymentAttributes `json:"attributes"`
}

type BrokerDeploymentAttributes struct {
	InstallID    string                 `json:"install_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ClientID     string                 `json:"client_id,omitempty"`
	ClientSecret string                 `json:"client_secret,omitempty"`
}

// BrokerDeploymentRequest represents a request to create/update a deployment
type BrokerDeploymentRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			BrokerAppInstalledInOrgID string                 `json:"broker_app_installed_in_org_id"`
			Metadata                  map[string]interface{} `json:"metadata,omitempty"`
		} `json:"attributes"`
	} `json:"data"`
}

// BrokerDeploymentResponse represents the response from deployment operations
type BrokerDeploymentResponse struct {
	Data BrokerDeployment `json:"data"`
}

// PaginationLinks represents JSON:API pagination links
type PaginationLinks struct {
	First string `json:"first,omitempty"`
	Last  string `json:"last,omitempty"`
	Prev  string `json:"prev,omitempty"`
	Next  string `json:"next,omitempty"`
	Self  string `json:"self,omitempty"`
}

// BrokerDeploymentsListResponse represents the response from listing deployments
type BrokerDeploymentsListResponse struct {
	Data  []BrokerDeployment `json:"data"`
	Links PaginationLinks    `json:"links,omitempty"`
}

// CreateBrokerDeployment creates a new broker deployment
func (c *Client) CreateBrokerDeployment(ctx context.Context, tenantID, installID, orgID string, metadata map[string]interface{}) (*BrokerDeploymentResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments?version=%s", tenantID, installID, c.apiVersion)

	req := BrokerDeploymentRequest{}
	req.Data.Type = "broker_deployment"
	req.Data.Attributes.BrokerAppInstalledInOrgID = orgID
	req.Data.Attributes.Metadata = metadata

	var resp BrokerDeploymentResponse
	if err := c.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create broker deployment: %w", err)
	}

	return &resp, nil
}

// GetBrokerDeployment retrieves a broker deployment by ID
// Note: The API only supports listing deployments, not getting a single one,
// so we list all deployments and filter by ID.
func (c *Client) GetBrokerDeployment(ctx context.Context, tenantID, installID, deploymentID string) (*BrokerDeploymentResponse, error) {
	// The API doesn't have a single-deployment GET endpoint, so we use the list endpoint and filter
	listResp, err := c.ListBrokerDeployments(ctx, tenantID, installID)
	if err != nil {
		return nil, fmt.Errorf("failed to get broker deployment: %w", err)
	}

	// Find the deployment with the matching ID
	for _, deployment := range listResp.Data {
		if deployment.ID == deploymentID {
			return &BrokerDeploymentResponse{Data: deployment}, nil
		}
	}

	// Return a 404-like error if not found
	return nil, &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("broker deployment %s not found", deploymentID),
	}
}

// ListBrokerDeployments lists all broker deployments for a tenant, handling pagination
func (c *Client) ListBrokerDeployments(ctx context.Context, tenantID, installID string) (*BrokerDeploymentsListResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments?version=%s", tenantID, installID, c.apiVersion)

	var allDeployments []BrokerDeployment

	for path != "" {
		var resp BrokerDeploymentsListResponse
		if err := c.Get(ctx, path, &resp); err != nil {
			return nil, fmt.Errorf("failed to list broker deployments: %w", err)
		}

		allDeployments = append(allDeployments, resp.Data...)

		// Check for next page
		if resp.Links.Next != "" {
			// The Next link is typically a full URL, extract the path
			path = resp.Links.Next
			// If it's a full URL, we need to strip the base URL
			if len(path) > len(c.baseURL) && path[:len(c.baseURL)] == c.baseURL {
				path = path[len(c.baseURL):]
			}
		} else {
			path = ""
		}
	}

	return &BrokerDeploymentsListResponse{Data: allDeployments}, nil
}

// ListBrokerDeploymentsForTenant lists all broker deployments for a tenant (without install ID), handling pagination
func (c *Client) ListBrokerDeploymentsForTenant(ctx context.Context, tenantID string) (*BrokerDeploymentsListResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/deployments?version=%s", tenantID, c.apiVersion)

	var allDeployments []BrokerDeployment

	for path != "" {
		var resp BrokerDeploymentsListResponse
		if err := c.Get(ctx, path, &resp); err != nil {
			return nil, fmt.Errorf("failed to list broker deployments for tenant: %w", err)
		}

		allDeployments = append(allDeployments, resp.Data...)

		// Check for next page
		if resp.Links.Next != "" {
			path = resp.Links.Next
			if len(path) > len(c.baseURL) && path[:len(c.baseURL)] == c.baseURL {
				path = path[len(c.baseURL):]
			}
		} else {
			path = ""
		}
	}

	return &BrokerDeploymentsListResponse{Data: allDeployments}, nil
}

// UpdateBrokerDeployment updates a broker deployment
func (c *Client) UpdateBrokerDeployment(ctx context.Context, tenantID, installID, deploymentID string, metadata map[string]interface{}) (*BrokerDeploymentResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s?version=%s", tenantID, installID, deploymentID, c.apiVersion)

	req := BrokerDeploymentRequest{}
	req.Data.Type = "broker_deployment"
	req.Data.Attributes.Metadata = metadata

	var resp BrokerDeploymentResponse
	if err := c.Patch(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to update broker deployment: %w", err)
	}

	return &resp, nil
}

// DeleteBrokerDeployment deletes a broker deployment
func (c *Client) DeleteBrokerDeployment(ctx context.Context, tenantID, installID, deploymentID string) error {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s?version=%s", tenantID, installID, deploymentID, c.apiVersion)

	if err := c.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete broker deployment: %w", err)
	}

	return nil
}

// BrokerCredential represents a broker credential reference
type BrokerCredential struct {
	ID         string                     `json:"id"`
	Type       string                     `json:"type"`
	Attributes BrokerCredentialAttributes `json:"attributes"`
}

type BrokerCredentialAttributes struct {
	Comment                 string `json:"comment,omitempty"`
	DeploymentID            string `json:"deployment_id,omitempty"`
	EnvironmentVariableName string `json:"environment_variable_name"`
	Type                    string `json:"type"`
}

// BrokerCredentialRequest represents a request to create credentials
type BrokerCredentialRequest struct {
	Data struct {
		Type       string                       `json:"type"`
		Attributes []BrokerCredentialAttributes `json:"attributes"`
	} `json:"data"`
}

// BrokerCredentialResponse represents the response from credential operations
// The API returns an array of credentials even when creating a single credential
type BrokerCredentialResponse struct {
	Data []BrokerCredential `json:"data"`
}

// BrokerCredentialsListResponse represents the response from listing credentials
type BrokerCredentialsListResponse struct {
	Data  []BrokerCredential `json:"data"`
	Links PaginationLinks    `json:"links,omitempty"`
}

// CreateBrokerCredential creates a new broker credential reference
func (c *Client) CreateBrokerCredential(ctx context.Context, tenantID, installID, deploymentID string, cred BrokerCredentialAttributes) (*BrokerCredentialResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/credentials?version=%s", tenantID, installID, deploymentID, c.apiVersion)

	req := BrokerCredentialRequest{}
	req.Data.Type = "deployment_credential"
	req.Data.Attributes = []BrokerCredentialAttributes{cred}

	var resp BrokerCredentialResponse
	if err := c.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create broker credential: %w", err)
	}

	return &resp, nil
}

// ListBrokerCredentials lists all credentials for a deployment, handling pagination
func (c *Client) ListBrokerCredentials(ctx context.Context, tenantID, installID, deploymentID string) (*BrokerCredentialsListResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/credentials?version=%s", tenantID, installID, deploymentID, c.apiVersion)

	var allCredentials []BrokerCredential

	for path != "" {
		var resp BrokerCredentialsListResponse
		if err := c.Get(ctx, path, &resp); err != nil {
			return nil, fmt.Errorf("failed to list broker credentials: %w", err)
		}

		allCredentials = append(allCredentials, resp.Data...)

		if resp.Links.Next != "" {
			path = resp.Links.Next
			if len(path) > len(c.baseURL) && path[:len(c.baseURL)] == c.baseURL {
				path = path[len(c.baseURL):]
			}
		} else {
			path = ""
		}
	}

	return &BrokerCredentialsListResponse{Data: allCredentials}, nil
}

// DeleteBrokerCredential deletes a broker credential
func (c *Client) DeleteBrokerCredential(ctx context.Context, tenantID, installID, deploymentID, credentialID string) error {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/credentials/%s?version=%s", tenantID, installID, deploymentID, credentialID, c.apiVersion)

	if err := c.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete broker credential: %w", err)
	}

	return nil
}

// BrokerConnection represents a broker connection
type BrokerConnection struct {
	ID         string                     `json:"id"`
	Type       string                     `json:"type"`
	Attributes BrokerConnectionAttributes `json:"attributes"`
}

type BrokerConnectionAttributes struct {
	DeploymentID  string                        `json:"deployment_id,omitempty"`
	Name          string                        `json:"name"`
	Configuration BrokerConnectionConfiguration `json:"configuration"`
	Secrets       *BrokerConnectionSecrets      `json:"secrets,omitempty"`
}

type BrokerConnectionConfiguration struct {
	Type     string                 `json:"type"`
	Required map[string]interface{} `json:"required"`
}

type BrokerConnectionSecrets struct {
	Primary   BrokerConnectionSecret `json:"primary"`
	Secondary BrokerConnectionSecret `json:"secondary"`
}

type BrokerConnectionSecret struct {
	Encrypted string `json:"encrypted"`
	ExpiresAt string `json:"expires_at"`
	Nonce     string `json:"nonce"`
}

// BrokerConnectionRequest represents a request to create/update a connection
type BrokerConnectionRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Name          string                        `json:"name"`
			Configuration BrokerConnectionConfiguration `json:"configuration"`
			DeploymentID  string                        `json:"deployment_id,omitempty"`
		} `json:"attributes"`
	} `json:"data"`
}

// BrokerConnectionResponse represents the response from connection operations
type BrokerConnectionResponse struct {
	Data BrokerConnection `json:"data"`
}

// BrokerConnectionsListResponse represents the response from listing connections
type BrokerConnectionsListResponse struct {
	Data  []BrokerConnection `json:"data"`
	Links PaginationLinks    `json:"links,omitempty"`
}

// CreateBrokerConnection creates a new broker connection
func (c *Client) CreateBrokerConnection(ctx context.Context, tenantID, installID, deploymentID, name string, config BrokerConnectionConfiguration) (*BrokerConnectionResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/connections?version=%s", tenantID, installID, deploymentID, c.apiVersion)

	req := BrokerConnectionRequest{}
	req.Data.Type = "broker_connection"
	req.Data.Attributes.Name = name
	req.Data.Attributes.Configuration = config
	req.Data.Attributes.DeploymentID = deploymentID

	var resp BrokerConnectionResponse
	if err := c.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create broker connection: %w", err)
	}

	return &resp, nil
}

// GetBrokerConnection retrieves a broker connection by ID
func (c *Client) GetBrokerConnection(ctx context.Context, tenantID, installID, deploymentID, connectionID string) (*BrokerConnectionResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/connections/%s?version=%s", tenantID, installID, deploymentID, connectionID, c.apiVersion)

	var resp BrokerConnectionResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get broker connection: %w", err)
	}

	return &resp, nil
}

// ListBrokerConnections lists all broker connections for a deployment, handling pagination
func (c *Client) ListBrokerConnections(ctx context.Context, tenantID, installID, deploymentID string) (*BrokerConnectionsListResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/connections?version=%s", tenantID, installID, deploymentID, c.apiVersion)

	var allConnections []BrokerConnection

	for path != "" {
		var resp BrokerConnectionsListResponse
		if err := c.Get(ctx, path, &resp); err != nil {
			return nil, fmt.Errorf("failed to list broker connections: %w", err)
		}

		allConnections = append(allConnections, resp.Data...)

		if resp.Links.Next != "" {
			path = resp.Links.Next
			if len(path) > len(c.baseURL) && path[:len(c.baseURL)] == c.baseURL {
				path = path[len(c.baseURL):]
			}
		} else {
			path = ""
		}
	}

	return &BrokerConnectionsListResponse{Data: allConnections}, nil
}

// ListBrokerConnectionsForOrg lists all broker connections for an organization, handling pagination
func (c *Client) ListBrokerConnectionsForOrg(ctx context.Context, orgID string) (*BrokerConnectionsListResponse, error) {
	path := fmt.Sprintf("/rest/orgs/%s/brokers/connections?version=%s", orgID, c.apiVersion)

	var allConnections []BrokerConnection

	for path != "" {
		var resp BrokerConnectionsListResponse
		if err := c.Get(ctx, path, &resp); err != nil {
			return nil, fmt.Errorf("failed to list broker connections for org: %w", err)
		}

		allConnections = append(allConnections, resp.Data...)

		if resp.Links.Next != "" {
			path = resp.Links.Next
			if len(path) > len(c.baseURL) && path[:len(c.baseURL)] == c.baseURL {
				path = path[len(c.baseURL):]
			}
		} else {
			path = ""
		}
	}

	return &BrokerConnectionsListResponse{Data: allConnections}, nil
}

// UpdateBrokerConnection updates a broker connection
func (c *Client) UpdateBrokerConnection(ctx context.Context, tenantID, installID, deploymentID, connectionID, name string, config BrokerConnectionConfiguration) (*BrokerConnectionResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/connections/%s?version=%s", tenantID, installID, deploymentID, connectionID, c.apiVersion)

	req := BrokerConnectionRequest{}
	req.Data.Type = "broker_connection"
	req.Data.Attributes.Name = name
	req.Data.Attributes.Configuration = config
	req.Data.Attributes.DeploymentID = deploymentID

	var resp BrokerConnectionResponse
	if err := c.Patch(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to update broker connection: %w", err)
	}

	return &resp, nil
}

// DeleteBrokerConnection deletes a broker connection
func (c *Client) DeleteBrokerConnection(ctx context.Context, tenantID, installID, deploymentID, connectionID string) error {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/connections/%s?version=%s", tenantID, installID, deploymentID, connectionID, c.apiVersion)

	if err := c.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete broker connection: %w", err)
	}

	return nil
}

// BrokerConnectionIntegration represents a connection integration
type BrokerConnectionIntegration struct {
	ID         string                                `json:"id"`
	Type       string                                `json:"type"`
	Attributes BrokerConnectionIntegrationAttributes `json:"attributes"`
}

type BrokerConnectionIntegrationAttributes struct {
	IntegrationID string `json:"integration_id"`
	OrgID         string `json:"org_id"`
	Type          string `json:"type"`
}

// BrokerConnectionIntegrationRequest represents a request to create an integration
type BrokerConnectionIntegrationRequest struct {
	Data struct {
		IntegrationID string `json:"integration_id,omitempty"`
		Type          string `json:"type"`
	} `json:"data"`
}

// BrokerConnectionIntegrationResponse represents the response from integration operations
type BrokerConnectionIntegrationResponse struct {
	Data BrokerConnectionIntegration `json:"data"`
}

// BrokerConnectionIntegrationsListResponse represents the response from listing integrations
type BrokerConnectionIntegrationsListResponse struct {
	Data []BrokerConnectionIntegration `json:"data"`
}

// CreateBrokerConnectionIntegration creates a new connection integration
func (c *Client) CreateBrokerConnectionIntegration(ctx context.Context, tenantID, connectionID, orgID, integrationID, integrationType string) (*BrokerConnectionIntegrationResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/connections/%s/orgs/%s/integration?version=%s", tenantID, connectionID, orgID, c.apiVersion)

	req := BrokerConnectionIntegrationRequest{}
	req.Data.IntegrationID = integrationID
	req.Data.Type = integrationType

	var resp BrokerConnectionIntegrationResponse
	if err := c.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create broker connection integration: %w", err)
	}

	return &resp, nil
}

// GetBrokerConnectionIntegrations retrieves all integrations for a connection
func (c *Client) GetBrokerConnectionIntegrations(ctx context.Context, tenantID, connectionID string) (*BrokerConnectionIntegrationsListResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/connections/%s/integrations?version=%s", tenantID, connectionID, c.apiVersion)

	var resp BrokerConnectionIntegrationsListResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get broker connection integrations: %w", err)
	}

	return &resp, nil
}

// DeleteBrokerConnectionIntegration deletes a connection integration
func (c *Client) DeleteBrokerConnectionIntegration(ctx context.Context, tenantID, connectionID, orgID, integrationID string) error {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/connections/%s/orgs/%s/integrations/%s?version=%s", tenantID, connectionID, orgID, integrationID, c.apiVersion)

	if err := c.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete broker connection integration: %w", err)
	}

	return nil
}

// BrokerMigrationOrg represents an organization available for migration
type BrokerMigrationOrg struct {
	ID         string                       `json:"id"`
	Type       string                       `json:"type"`
	Attributes BrokerMigrationOrgAttributes `json:"attributes"`
}

type BrokerMigrationOrgAttributes struct {
	OrgID   string `json:"org_id"`
	OrgName string `json:"org_name"`
}

// BrokerMigrationOrgsListResponse represents the response from listing migration orgs
type BrokerMigrationOrgsListResponse struct {
	Data  []BrokerMigrationOrg `json:"data"`
	Links PaginationLinks      `json:"links,omitempty"`
}

// ListBrokerMigrationOrgs lists organizations available for bulk migration, handling pagination
func (c *Client) ListBrokerMigrationOrgs(ctx context.Context, tenantID, installID, deploymentID, connectionID string) (*BrokerMigrationOrgsListResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/connections/%s/bulk_migration?version=%s", tenantID, installID, deploymentID, connectionID, c.apiVersion)

	var allOrgs []BrokerMigrationOrg

	for path != "" {
		var resp BrokerMigrationOrgsListResponse
		if err := c.Get(ctx, path, &resp); err != nil {
			return nil, fmt.Errorf("failed to list broker migration orgs: %w", err)
		}

		allOrgs = append(allOrgs, resp.Data...)

		if resp.Links.Next != "" {
			path = resp.Links.Next
			if len(path) > len(c.baseURL) && path[:len(c.baseURL)] == c.baseURL {
				path = path[len(c.baseURL):]
			}
		} else {
			path = ""
		}
	}

	return &BrokerMigrationOrgsListResponse{Data: allOrgs}, nil
}

// BrokerBulkMigrationRequest represents a request to perform bulk migration
type BrokerBulkMigrationRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			OrgIDs []string `json:"org_ids"`
		} `json:"attributes"`
	} `json:"data"`
}

// BrokerBulkMigrationResponse represents the response from bulk migration
type BrokerBulkMigrationResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Status string `json:"status"`
		} `json:"attributes"`
	} `json:"data"`
}

// CreateBrokerBulkMigration performs bulk migration of orgs to universal broker
func (c *Client) CreateBrokerBulkMigration(ctx context.Context, tenantID, installID, deploymentID, connectionID string, orgIDs []string) (*BrokerBulkMigrationResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/connections/%s/bulk_migration?version=%s", tenantID, installID, deploymentID, connectionID, c.apiVersion)

	req := BrokerBulkMigrationRequest{}
	req.Data.Type = "broker_bulk_migration"
	req.Data.Attributes.OrgIDs = orgIDs

	var resp BrokerBulkMigrationResponse
	if err := c.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create broker bulk migration: %w", err)
	}

	return &resp, nil
}

// BrokerContext represents a broker context
type BrokerContext struct {
	ID            string                  `json:"id"`
	Type          string                  `json:"type"`
	Attributes    BrokerContextAttributes `json:"attributes"`
	Relationships *BrokerContextRelationships `json:"relationships,omitempty"`
}

type BrokerContextAttributes struct {
	Context map[string]string `json:"context"`
}

type BrokerContextRelationships struct {
	BrokerConnections   []BrokerContextRelationship `json:"broker_connections,omitempty"`
	AppliedIntegrations []BrokerContextIntegrationRelationship `json:"applied_integrations,omitempty"`
}

type BrokerContextRelationship struct {
	Data struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"data"`
}

type BrokerContextIntegrationRelationship struct {
	Data struct {
		ID    string `json:"id"`
		OrgID string `json:"org_id"`
		Type  string `json:"type"`
	} `json:"data"`
}

// BrokerContextRequest represents a request to create/update a context
type BrokerContextRequest struct {
	Data struct {
		ID         string `json:"id,omitempty"`
		Type       string `json:"type"`
		Attributes struct {
			Context      map[string]string `json:"context"`
			ConnectionID string            `json:"connection_id,omitempty"`
		} `json:"attributes"`
	} `json:"data"`
}

// BrokerContextResponse represents the response from context operations
type BrokerContextResponse struct {
	Data BrokerContext `json:"data"`
}

// BrokerContextsListResponse represents the response from listing contexts
type BrokerContextsListResponse struct {
	Data  []BrokerContext `json:"data"`
	Links PaginationLinks `json:"links,omitempty"`
}

// CreateBrokerContext creates a new broker context
func (c *Client) CreateBrokerContext(ctx context.Context, tenantID, installID, deploymentID, connectionID string, contextData map[string]string) (*BrokerContextResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/contexts?version=%s", tenantID, installID, deploymentID, c.apiVersion)

	req := BrokerContextRequest{}
	req.Data.Type = "broker_context"
	req.Data.Attributes.Context = contextData
	req.Data.Attributes.ConnectionID = connectionID

	var resp BrokerContextResponse
	if err := c.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create broker context: %w", err)
	}

	return &resp, nil
}

// GetBrokerContext retrieves a broker context by ID
func (c *Client) GetBrokerContext(ctx context.Context, tenantID, installID, contextID string) (*BrokerContextResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/contexts/%s?version=%s", tenantID, installID, contextID, c.apiVersion)

	var resp BrokerContextResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get broker context: %w", err)
	}

	return &resp, nil
}

// UpdateBrokerContext updates a broker context
func (c *Client) UpdateBrokerContext(ctx context.Context, tenantID, installID, contextID string, contextData map[string]string) (*BrokerContextResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/contexts/%s?version=%s", tenantID, installID, contextID, c.apiVersion)

	req := BrokerContextRequest{}
	req.Data.ID = contextID
	req.Data.Type = "broker_context"
	req.Data.Attributes.Context = contextData

	var resp BrokerContextResponse
	if err := c.Patch(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to update broker context: %w", err)
	}

	return &resp, nil
}

// DeleteBrokerContext deletes a broker context
func (c *Client) DeleteBrokerContext(ctx context.Context, tenantID, installID, contextID string) error {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/contexts/%s?version=%s", tenantID, installID, contextID, c.apiVersion)

	if err := c.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete broker context: %w", err)
	}

	return nil
}

// ListConnectionContexts lists all broker contexts for a connection, handling pagination
func (c *Client) ListConnectionContexts(ctx context.Context, tenantID, installID, connectionID string) (*BrokerContextsListResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/connections/%s/contexts?version=%s", tenantID, installID, connectionID, c.apiVersion)

	var allContexts []BrokerContext

	for path != "" {
		var resp BrokerContextsListResponse
		if err := c.Get(ctx, path, &resp); err != nil {
			return nil, fmt.Errorf("failed to list broker contexts for connection: %w", err)
		}

		allContexts = append(allContexts, resp.Data...)

		if resp.Links.Next != "" {
			path = resp.Links.Next
			if len(path) > len(c.baseURL) && path[:len(c.baseURL)] == c.baseURL {
				path = path[len(c.baseURL):]
			}
		} else {
			path = ""
		}
	}

	return &BrokerContextsListResponse{Data: allContexts}, nil
}

// ListDeploymentContexts lists all broker contexts for a deployment, handling pagination
func (c *Client) ListDeploymentContexts(ctx context.Context, tenantID, installID, deploymentID string) (*BrokerContextsListResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/deployments/%s/contexts?version=%s", tenantID, installID, deploymentID, c.apiVersion)

	var allContexts []BrokerContext

	for path != "" {
		var resp BrokerContextsListResponse
		if err := c.Get(ctx, path, &resp); err != nil {
			return nil, fmt.Errorf("failed to list broker contexts for deployment: %w", err)
		}

		allContexts = append(allContexts, resp.Data...)

		if resp.Links.Next != "" {
			path = resp.Links.Next
			if len(path) > len(c.baseURL) && path[:len(c.baseURL)] == c.baseURL {
				path = path[len(c.baseURL):]
			}
		} else {
			path = ""
		}
	}

	return &BrokerContextsListResponse{Data: allContexts}, nil
}

// BrokerContextIntegrationRequest represents a request to associate an integration with a context
type BrokerContextIntegrationRequest struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			OrgID string `json:"org_id"`
		} `json:"attributes"`
	} `json:"data"`
}

// BrokerContextIntegrationResponse represents the response from context integration operations
type BrokerContextIntegrationResponse struct {
	Data struct {
		ID            string `json:"id"`
		Type          string `json:"type"`
		Relationships struct {
			IntegrationsRelationships []struct {
				Data struct {
					ID              string `json:"id"`
					OrgID           string `json:"org_id"`
					IntegrationType string `json:"integration_type"`
					Type            string `json:"type"`
				} `json:"data"`
			} `json:"integrations_relationships"`
		} `json:"relationships"`
	} `json:"data"`
}

// UpdateBrokerContextIntegration associates an integration with a broker context
func (c *Client) UpdateBrokerContextIntegration(ctx context.Context, tenantID, installID, contextID, integrationID, orgID string) (*BrokerContextIntegrationResponse, error) {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/contexts/%s/integration?version=%s", tenantID, installID, contextID, c.apiVersion)

	req := BrokerContextIntegrationRequest{}
	req.Data.ID = integrationID
	req.Data.Type = "broker_integration"
	req.Data.Attributes.OrgID = orgID

	var resp BrokerContextIntegrationResponse
	if err := c.Patch(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to update broker context integration: %w", err)
	}

	return &resp, nil
}

// DeleteBrokerContextIntegration removes an integration association from a broker context
func (c *Client) DeleteBrokerContextIntegration(ctx context.Context, tenantID, installID, contextID, integrationID string) error {
	path := fmt.Sprintf("/rest/tenants/%s/brokers/installs/%s/contexts/%s/integrations/%s?version=%s", tenantID, installID, contextID, integrationID, c.apiVersion)

	if err := c.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete broker context integration: %w", err)
	}

	return nil
}
