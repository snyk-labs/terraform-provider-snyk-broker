// Copyright (c) Snyk Ltd.

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/snyk-labs/snyk-broker-provider/internal/client"
	"github.com/snyk-labs/snyk-broker-provider/internal/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BrokerDeploymentResource{}
var _ resource.ResourceWithImportState = &BrokerDeploymentResource{}

// NewBrokerDeploymentResource creates a new resource instance
func NewBrokerDeploymentResource() resource.Resource {
	return &BrokerDeploymentResource{}
}

// BrokerDeploymentResource defines the resource implementation.
type BrokerDeploymentResource struct {
	client *client.Client
}

// BrokerDeploymentResourceModel describes the resource data model.
type BrokerDeploymentResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TenantID     types.String `tfsdk:"tenant_id"`
	InstallID    types.String `tfsdk:"install_id"`
	OrgID        types.String `tfsdk:"org_id"`
	Name         types.String `tfsdk:"name"`
	Metadata     types.Map    `tfsdk:"metadata"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func (r *BrokerDeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_deployment"
}

func (r *BrokerDeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Snyk Broker deployment. A deployment represents a running Broker instance that can have multiple connections.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this deployment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID for the deployment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"install_id": schema.StringAttribute{
				Description: "The Broker app installation ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID where the Broker app is installed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A human-readable name for the deployment.",
				Required:    true,
			},
			"metadata": schema.MapAttribute{
				Description: "Additional metadata for the deployment (e.g., cluster information).",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"client_id": schema.StringAttribute{
				Description: "The OAuth client ID for running the Broker client.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_secret": schema.StringAttribute{
				Description: "The OAuth client secret for running the Broker client.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *BrokerDeploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*common.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = providerData.Client
}

func (r *BrokerDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BrokerDeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert metadata to map[string]interface{}
	metadata := make(map[string]interface{})
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		var metadataMap map[string]string
		resp.Diagnostics.Append(data.Metadata.ElementsAs(ctx, &metadataMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Add the deployment name to metadata
		metadata["deployment_name"] = data.Name.ValueString()
		for k, v := range metadataMap {
			metadata[k] = v
		}
	} else {
		metadata["deployment_name"] = data.Name.ValueString()
	}

	tflog.Debug(ctx, "Creating broker deployment", map[string]interface{}{
		"tenant_id":  data.TenantID.ValueString(),
		"install_id": data.InstallID.ValueString(),
		"name":       data.Name.ValueString(),
	})

	result, err := r.client.CreateBrokerDeployment(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.OrgID.ValueString(),
		metadata,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Broker deployment", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.ClientID = types.StringValue(result.Data.Attributes.ClientID)
	data.ClientSecret = types.StringValue(result.Data.Attributes.ClientSecret)

	tflog.Trace(ctx, "Created broker deployment", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BrokerDeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker deployment", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	result, err := r.client.GetBrokerDeployment(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		// Check if it's a 404 (use errors.As for wrapped errors)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Broker deployment", err.Error())
		return
	}

	// Update state from API response
	data.ID = types.StringValue(result.Data.ID)

	// Extract deployment_name from metadata if present
	if name, ok := result.Data.Attributes.Metadata["deployment_name"].(string); ok {
		data.Name = types.StringValue(name)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BrokerDeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert metadata to map[string]interface{}
	metadata := make(map[string]interface{})
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		var metadataMap map[string]string
		resp.Diagnostics.Append(data.Metadata.ElementsAs(ctx, &metadataMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		metadata["deployment_name"] = data.Name.ValueString()
		for k, v := range metadataMap {
			metadata[k] = v
		}
	} else {
		metadata["deployment_name"] = data.Name.ValueString()
	}

	tflog.Debug(ctx, "Updating broker deployment", map[string]interface{}{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
	})

	_, err := r.client.UpdateBrokerDeployment(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ID.ValueString(),
		metadata,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Broker deployment", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BrokerDeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting broker deployment", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	// Step 1: Delete all connections for this deployment first
	// Connections must be deleted before credentials because they may reference credentials
	connections, err := r.client.ListBrokerConnections(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		// Use errors.As for wrapped errors
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
			tflog.Warn(ctx, "Failed to list connections for deployment, attempting to continue", map[string]interface{}{
				"error": err.Error(),
			})
		}
	} else if connections != nil {
		for _, conn := range connections.Data {
			tflog.Debug(ctx, "Deleting connection before deployment", map[string]interface{}{
				"connection_id": conn.ID,
			})

			// First delete any integrations for this connection
			integrations, err := r.client.GetBrokerConnectionIntegrations(
				ctx,
				data.TenantID.ValueString(),
				conn.ID,
			)
			if err == nil && integrations != nil {
				for _, integration := range integrations.Data {
					// Use the top-level ID if IntegrationID attribute is empty
					integrationID := integration.Attributes.IntegrationID
					if integrationID == "" {
						integrationID = integration.ID
					}

					tflog.Debug(ctx, "Deleting integration before connection", map[string]interface{}{
						"integration_id": integrationID,
						"org_id":         integration.Attributes.OrgID,
					})

					err := r.client.DeleteBrokerConnectionIntegration(
						ctx,
						data.TenantID.ValueString(),
						conn.ID,
						integration.Attributes.OrgID,
						integrationID,
					)
					if err != nil {
						// Use errors.As for wrapped errors
						var apiErr *client.APIError
						if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
							continue
						}
						resp.Diagnostics.AddError(
							"Failed to delete integration before deployment",
							fmt.Sprintf("Integration %s: %s", integrationID, err.Error()),
						)
						return
					}
				}
			}

			// Now delete the connection
			err = r.client.DeleteBrokerConnection(
				ctx,
				data.TenantID.ValueString(),
				data.InstallID.ValueString(),
				data.ID.ValueString(),
				conn.ID,
			)
			if err != nil {
				// Use errors.As for wrapped errors
				var apiErr *client.APIError
				if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
					continue
				}
				resp.Diagnostics.AddError(
					"Failed to delete connection before deployment",
					fmt.Sprintf("Connection %s: %s", conn.ID, err.Error()),
				)
				return
			}
		}
	}

	// Step 2: Delete all credentials for this deployment
	credentials, err := r.client.ListBrokerCredentials(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		// Use errors.As for wrapped errors
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
			tflog.Warn(ctx, "Failed to list credentials for deployment, attempting to continue", map[string]interface{}{
				"error": err.Error(),
			})
		}
	} else if credentials != nil {
		for _, cred := range credentials.Data {
			tflog.Debug(ctx, "Deleting credential before deployment", map[string]interface{}{
				"credential_id": cred.ID,
			})

			err := r.client.DeleteBrokerCredential(
				ctx,
				data.TenantID.ValueString(),
				data.InstallID.ValueString(),
				data.ID.ValueString(),
				cred.ID,
			)
			if err != nil {
				// Use errors.As for wrapped errors
				var apiErr *client.APIError
				if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
					continue
				}
				resp.Diagnostics.AddError(
					"Failed to delete credential before deployment",
					fmt.Sprintf("Credential %s: %s", cred.ID, err.Error()),
				)
				return
			}
		}
	}

	// Step 3: Delete the deployment itself
	err = r.client.DeleteBrokerDeployment(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		// Check if it's a 404 - already deleted (use errors.As for wrapped errors)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Failed to delete Broker deployment", err.Error())
		return
	}
}

func (r *BrokerDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: tenant_id:install_id:deployment_id
	resp.Diagnostics.AddError(
		"Import Not Fully Supported",
		"Broker deployment import requires tenant_id:install_id:deployment_id format. "+
			"Note that client_id and client_secret cannot be retrieved after initial creation.",
	)
}
