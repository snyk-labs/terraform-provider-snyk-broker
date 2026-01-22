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

	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
	"github.com/snyk/terraform-provider-snyk-broker/internal/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BrokerConnectionIntegrationResource{}
var _ resource.ResourceWithImportState = &BrokerConnectionIntegrationResource{}

// NewBrokerConnectionIntegrationResource creates a new resource instance
func NewBrokerConnectionIntegrationResource() resource.Resource {
	return &BrokerConnectionIntegrationResource{}
}

// BrokerConnectionIntegrationResource defines the resource implementation.
type BrokerConnectionIntegrationResource struct {
	client *client.Client
}

// BrokerConnectionIntegrationResourceModel describes the resource data model.
type BrokerConnectionIntegrationResourceModel struct {
	ID            types.String `tfsdk:"id"`
	TenantID      types.String `tfsdk:"tenant_id"`
	ConnectionID  types.String `tfsdk:"connection_id"`
	OrgID         types.String `tfsdk:"org_id"`
	IntegrationID types.String `tfsdk:"integration_id"`
	Type          types.String `tfsdk:"type"`
}

func (r *BrokerConnectionIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_connection_integration"
}

func (r *BrokerConnectionIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Links a Snyk organization integration to a Broker connection. This enables the organization to use the Broker for the specified integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this integration link.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connection_id": schema.StringAttribute{
				Description: "The Broker connection ID to link to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID that owns the integration.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"integration_id": schema.StringAttribute{
				Description: "The integration ID within the organization. If not provided, a new integration will be created.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The integration type (e.g., github, gitlab, bitbucket-server).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *BrokerConnectionIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BrokerConnectionIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BrokerConnectionIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating broker connection integration", map[string]interface{}{
		"connection_id":  data.ConnectionID.ValueString(),
		"org_id":         data.OrgID.ValueString(),
		"integration_id": data.IntegrationID.ValueString(),
		"type":           data.Type.ValueString(),
	})

	// Get the integration ID value (may be empty if not provided)
	integrationID := data.IntegrationID.ValueString()

	result, err := r.client.CreateBrokerConnectionIntegration(
		ctx,
		data.TenantID.ValueString(),
		data.ConnectionID.ValueString(),
		data.OrgID.ValueString(),
		integrationID,
		data.Type.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Broker connection integration", err.Error())
		return
	}

	// Update integration_id from the API response
	// Use the top-level ID if IntegrationID attribute is empty (fallback pattern matching broker_connection.go)
	if result.Data.Attributes.IntegrationID != "" {
		data.IntegrationID = types.StringValue(result.Data.Attributes.IntegrationID)
	} else if result.Data.ID != "" {
		// If API didn't return an integration_id in attributes, use the top-level ID
		data.IntegrationID = types.StringValue(result.Data.ID)
	} else if !data.IntegrationID.IsUnknown() && !data.IntegrationID.IsNull() {
		// Keep the user's provided value (already set)
	} else {
		// This shouldn't happen, but set to empty string as last resort
		data.IntegrationID = types.StringValue("")
	}

	// Use a composite ID since the API might not return a unique ID
	if result.Data.ID != "" {
		data.ID = types.StringValue(result.Data.ID)
	} else {
		data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", data.ConnectionID.ValueString(), data.OrgID.ValueString(), data.IntegrationID.ValueString()))
	}

	tflog.Trace(ctx, "Created broker connection integration", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerConnectionIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BrokerConnectionIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker connection integration", map[string]interface{}{
		"id":             data.ID.ValueString(),
		"org_id":         data.OrgID.ValueString(),
		"integration_id": data.IntegrationID.ValueString(),
	})

	// List all integrations for the connection and find ours
	result, err := r.client.GetBrokerConnectionIntegrations(
		ctx,
		data.TenantID.ValueString(),
		data.ConnectionID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Broker connection integrations", err.Error())
		return
	}

	// Find the integration matching our resource
	// We use multiple matching strategies because the API may not return all attributes:
	// 1. Match by ID (most reliable since top-level ID is always populated)
	// 2. Match by org_id and integration_id (when attributes are populated)
	// 3. Match by org_id only (fallback when integration_id is unknown)
	found := false
	for _, integration := range result.Data {
		// Use the top-level ID if IntegrationID attribute is empty (fallback pattern)
		apiIntegrationID := integration.Attributes.IntegrationID
		if apiIntegrationID == "" {
			apiIntegrationID = integration.ID
		}

		// Strategy 1: Match by ID (the top-level ID returned by API)
		// This is the most reliable match since the API always returns the ID correctly
		idMatches := integration.ID == data.ID.ValueString()

		// Strategy 2: Match by org_id and integration_id (when API attributes are populated)
		orgMatches := integration.Attributes.OrgID != "" && integration.Attributes.OrgID == data.OrgID.ValueString()
		integrationMatches := data.IntegrationID.IsNull() ||
			data.IntegrationID.IsUnknown() ||
			data.IntegrationID.ValueString() == "" ||
			apiIntegrationID == data.IntegrationID.ValueString()

		if idMatches || (orgMatches && integrationMatches) {
			found = true
			tflog.Debug(ctx, "Found matching integration", map[string]interface{}{
				"api_id":             integration.ID,
				"api_org_id":         integration.Attributes.OrgID,
				"api_integration_id": integration.Attributes.IntegrationID,
				"api_type":           integration.Attributes.Type,
				"matched_by_id":      idMatches,
			})
			// Only update attributes from API if they are non-empty
			// Keep existing state values when API returns empty attributes
			if integration.Attributes.Type != "" {
				data.Type = types.StringValue(integration.Attributes.Type)
			}
			// Update the integration_id from the API response using the fallback pattern
			if apiIntegrationID != "" {
				data.IntegrationID = types.StringValue(apiIntegrationID)
			}
			break
		}
	}

	if !found {
		tflog.Warn(ctx, "Integration not found in API response, marking as deleted", map[string]interface{}{
			"id":               data.ID.ValueString(),
			"org_id":           data.OrgID.ValueString(),
			"integrations_len": len(result.Data),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerConnectionIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Connection integrations are immutable - all changes require replacement
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Broker connection integrations cannot be updated. Changes require replacement.",
	)
}

func (r *BrokerConnectionIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BrokerConnectionIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the ID as fallback if integration_id is empty (for backwards compatibility with existing state)
	integrationID := data.IntegrationID.ValueString()
	if integrationID == "" {
		integrationID = data.ID.ValueString()
	}

	tflog.Debug(ctx, "Deleting broker connection integration", map[string]interface{}{
		"id":             data.ID.ValueString(),
		"integration_id": integrationID,
	})

	err := r.client.DeleteBrokerConnectionIntegration(
		ctx,
		data.TenantID.ValueString(),
		data.ConnectionID.ValueString(),
		data.OrgID.ValueString(),
		integrationID,
	)
	if err != nil {
		// Check if it's a 404 - already deleted (use errors.As for wrapped errors)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Failed to delete Broker connection integration", err.Error())
		return
	}
}

func (r *BrokerConnectionIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import Not Supported",
		"Broker connection integrations cannot be imported. Please create a new integration link.",
	)
}
