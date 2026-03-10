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
var _ resource.Resource = &BrokerContextIntegrationResource{}

// NewBrokerContextIntegrationResource creates a new resource instance
func NewBrokerContextIntegrationResource() resource.Resource {
	return &BrokerContextIntegrationResource{}
}

// BrokerContextIntegrationResource defines the resource implementation.
type BrokerContextIntegrationResource struct {
	client *client.Client
}

// BrokerContextIntegrationResourceModel describes the resource data model.
type BrokerContextIntegrationResourceModel struct {
	ID            types.String `tfsdk:"id"`
	TenantID      types.String `tfsdk:"tenant_id"`
	InstallID     types.String `tfsdk:"install_id"`
	ContextID     types.String `tfsdk:"context_id"`
	IntegrationID types.String `tfsdk:"integration_id"`
	OrgID         types.String `tfsdk:"org_id"`
}

func (r *BrokerContextIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_context_integration"
}

func (r *BrokerContextIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the association between a Snyk Broker context and an integration. This allows you to apply a broker context to a specific organization integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this context integration association (same as integration_id).",
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
			"install_id": schema.StringAttribute{
				Description: "The Broker app installation ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"context_id": schema.StringAttribute{
				Description: "The broker context ID to associate the integration with.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"integration_id": schema.StringAttribute{
				Description: "The integration ID to associate with the context.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID that the integration belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *BrokerContextIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BrokerContextIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BrokerContextIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating broker context integration", map[string]interface{}{
		"context_id":     data.ContextID.ValueString(),
		"integration_id": data.IntegrationID.ValueString(),
		"org_id":         data.OrgID.ValueString(),
	})

	_, err := r.client.UpdateBrokerContextIntegration(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ContextID.ValueString(),
		data.IntegrationID.ValueString(),
		data.OrgID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Broker context integration", err.Error())
		return
	}

	data.ID = data.IntegrationID

	tflog.Trace(ctx, "Created broker context integration", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerContextIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BrokerContextIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker context integration", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"context_id": data.ContextID.ValueString(),
	})

	result, err := r.client.GetBrokerContext(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ContextID.ValueString(),
	)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Broker context for integration", err.Error())
		return
	}

	found := false
	if result.Data.Relationships != nil {
		for _, integration := range result.Data.Relationships.AppliedIntegrations {
			if integration.Data.ID == data.IntegrationID.ValueString() {
				found = true
				break
			}
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerContextIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Broker context integrations cannot be updated in place. All attributes require replacement.",
	)
}

func (r *BrokerContextIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BrokerContextIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting broker context integration", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"context_id": data.ContextID.ValueString(),
	})

	err := r.client.DeleteBrokerContextIntegration(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ContextID.ValueString(),
		data.IntegrationID.ValueString(),
	)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Failed to delete Broker context integration", err.Error())
		return
	}
}
