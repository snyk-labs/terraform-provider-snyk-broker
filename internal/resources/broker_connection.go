// Copyright (c) Snyk Ltd.

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/snyk-labs/snyk-broker-provider/internal/client"
	"github.com/snyk-labs/snyk-broker-provider/internal/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BrokerConnectionResource{}
var _ resource.ResourceWithImportState = &BrokerConnectionResource{}

// NewBrokerConnectionResource creates a new resource instance
func NewBrokerConnectionResource() resource.Resource {
	return &BrokerConnectionResource{}
}

// BrokerConnectionResource defines the resource implementation.
type BrokerConnectionResource struct {
	client *client.Client
}

// BrokerConnectionResourceModel describes the resource data model.
type BrokerConnectionResourceModel struct {
	ID            types.String `tfsdk:"id"`
	TenantID      types.String `tfsdk:"tenant_id"`
	InstallID     types.String `tfsdk:"install_id"`
	DeploymentID  types.String `tfsdk:"deployment_id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	Configuration types.Map    `tfsdk:"configuration"`
}

func (r *BrokerConnectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_connection"
}

func (r *BrokerConnectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Snyk Broker connection. Connections define how the Broker client connects to your SCM, container registry, or other integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this connection.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID for the connection.",
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
			"deployment_id": schema.StringAttribute{
				Description: "The deployment ID this connection belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A human-readable name for the connection.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of connection (e.g., github, github-enterprise, gitlab, bitbucket-server, azure-repos, jira, artifactory, nexus, container-registry-agent).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration": schema.MapAttribute{
				Description: "The connection configuration. Required keys depend on the connection type. Typically includes credential references and URLs.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *BrokerConnectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BrokerConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BrokerConnectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert configuration to map[string]interface{}
	configMap := make(map[string]interface{})
	if !data.Configuration.IsNull() && !data.Configuration.IsUnknown() {
		var configStrMap map[string]string
		resp.Diagnostics.Append(data.Configuration.ElementsAs(ctx, &configStrMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for k, v := range configStrMap {
			configMap[k] = v
		}
	}

	config := client.BrokerConnectionConfiguration{
		Type:     data.Type.ValueString(),
		Required: configMap,
	}

	tflog.Debug(ctx, "Creating broker connection", map[string]interface{}{
		"deployment_id": data.DeploymentID.ValueString(),
		"name":          data.Name.ValueString(),
		"type":          data.Type.ValueString(),
	})

	result, err := r.client.CreateBrokerConnection(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		data.Name.ValueString(),
		config,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Broker connection", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)

	tflog.Trace(ctx, "Created broker connection", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BrokerConnectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker connection", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	result, err := r.client.GetBrokerConnection(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		// Check if it's a 404 (use errors.As for wrapped errors)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Broker connection", err.Error())
		return
	}

	// Update state from API response
	data.ID = types.StringValue(result.Data.ID)
	data.Name = types.StringValue(result.Data.Attributes.Name)
	data.Type = types.StringValue(result.Data.Attributes.Configuration.Type)

	// Convert configuration back to map
	configMap := make(map[string]string)
	for k, v := range result.Data.Attributes.Configuration.Required {
		if strVal, ok := v.(string); ok {
			configMap[k] = strVal
		}
	}
	configMapValue, diags := types.MapValueFrom(ctx, types.StringType, configMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Configuration = configMapValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BrokerConnectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert configuration to map[string]interface{}
	configMap := make(map[string]interface{})
	if !data.Configuration.IsNull() && !data.Configuration.IsUnknown() {
		var configStrMap map[string]string
		resp.Diagnostics.Append(data.Configuration.ElementsAs(ctx, &configStrMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for k, v := range configStrMap {
			configMap[k] = v
		}
	}

	config := client.BrokerConnectionConfiguration{
		Type:     data.Type.ValueString(),
		Required: configMap,
	}

	tflog.Debug(ctx, "Updating broker connection", map[string]interface{}{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
	})

	_, err := r.client.UpdateBrokerConnection(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		data.ID.ValueString(),
		data.Name.ValueString(),
		config,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Broker connection", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BrokerConnectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting broker connection", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	// First, delete any integrations associated with this connection
	// This ensures we can delete the connection even if integrations exist
	integrations, err := r.client.GetBrokerConnectionIntegrations(
		ctx,
		data.TenantID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		// If we can't list integrations, try to delete the connection anyway
		// It might fail, but let's not block on listing issues (use errors.As for wrapped errors)
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
			tflog.Warn(ctx, "Failed to list integrations for connection, attempting delete anyway", map[string]interface{}{
				"error": err.Error(),
			})
		}
	} else if integrations != nil {
		// Track integrations we couldn't delete due to missing org_id
		var integrationsWithMissingOrgID []string

		for _, integration := range integrations.Data {
			// Use the top-level ID if IntegrationID attribute is empty
			integrationID := integration.Attributes.IntegrationID
			if integrationID == "" {
				integrationID = integration.ID
			}

			// Check if org_id is available - the API sometimes returns empty attributes
			orgID := integration.Attributes.OrgID
			if orgID == "" {
				tflog.Warn(ctx, "Integration has empty org_id from API, cannot delete via cascade", map[string]interface{}{
					"integration_id": integrationID,
					"api_id":         integration.ID,
				})
				integrationsWithMissingOrgID = append(integrationsWithMissingOrgID, integrationID)
				continue
			}

			tflog.Debug(ctx, "Deleting integration before connection", map[string]interface{}{
				"integration_id": integrationID,
				"org_id":         orgID,
			})

			err := r.client.DeleteBrokerConnectionIntegration(
				ctx,
				data.TenantID.ValueString(),
				data.ID.ValueString(),
				orgID,
				integrationID,
			)
			if err != nil {
				// Check if it's a 404 - already deleted (use errors.As for wrapped errors)
				var apiErr *client.APIError
				if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
					continue
				}
				resp.Diagnostics.AddError(
					"Failed to delete integration before connection",
					fmt.Sprintf("Integration %s: %s", integrationID, err.Error()),
				)
				return
			}
		}

		// If we had integrations we couldn't delete, warn the user
		if len(integrationsWithMissingOrgID) > 0 {
			tflog.Warn(ctx, "Some integrations could not be deleted due to missing org_id in API response. "+
				"These integrations should be managed by snyk_broker_connection_integration resources.", map[string]interface{}{
				"integration_ids": integrationsWithMissingOrgID,
			})
		}
	}

	// Now delete the connection itself
	err = r.client.DeleteBrokerConnection(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		// Check if it's a 404 - already deleted (use errors.As for wrapped errors)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Failed to delete Broker connection", err.Error())
		return
	}
}

func (r *BrokerConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import Not Supported",
		"Broker connections cannot be imported. Please create a new connection.",
	)
}
