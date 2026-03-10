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
var _ resource.Resource = &BrokerContextResource{}
var _ resource.ResourceWithImportState = &BrokerContextResource{}

// NewBrokerContextResource creates a new resource instance
func NewBrokerContextResource() resource.Resource {
	return &BrokerContextResource{}
}

// BrokerContextResource defines the resource implementation.
type BrokerContextResource struct {
	client *client.Client
}

// BrokerContextResourceModel describes the resource data model.
type BrokerContextResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TenantID     types.String `tfsdk:"tenant_id"`
	InstallID    types.String `tfsdk:"install_id"`
	DeploymentID types.String `tfsdk:"deployment_id"`
	ConnectionID types.String `tfsdk:"connection_id"`
	Context      types.Map    `tfsdk:"context"`
}

func (r *BrokerContextResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_context"
}

func (r *BrokerContextResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Snyk Broker context. Contexts provide environment-specific configuration values that can be applied to broker connections.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this context.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID for the context.",
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
				Description: "The deployment ID this context belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connection_id": schema.StringAttribute{
				Description: "The connection ID this context is associated with.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"context": schema.MapAttribute{
				Description: "The context configuration as key-value pairs. These values provide environment-specific configuration for the broker connection.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *BrokerContextResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BrokerContextResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BrokerContextResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	contextMap := make(map[string]string)
	if !data.Context.IsNull() && !data.Context.IsUnknown() {
		resp.Diagnostics.Append(data.Context.ElementsAs(ctx, &contextMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Debug(ctx, "Creating broker context", map[string]interface{}{
		"deployment_id": data.DeploymentID.ValueString(),
		"connection_id": data.ConnectionID.ValueString(),
	})

	result, err := r.client.CreateBrokerContext(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		data.ConnectionID.ValueString(),
		contextMap,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Broker context", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)

	tflog.Trace(ctx, "Created broker context", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerContextResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BrokerContextResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker context", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	result, err := r.client.GetBrokerContext(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Broker context", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)

	contextMapValue, diags := types.MapValueFrom(ctx, types.StringType, result.Data.Attributes.Context)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Context = contextMapValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerContextResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BrokerContextResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	contextMap := make(map[string]string)
	if !data.Context.IsNull() && !data.Context.IsUnknown() {
		resp.Diagnostics.Append(data.Context.ElementsAs(ctx, &contextMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Debug(ctx, "Updating broker context", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	_, err := r.client.UpdateBrokerContext(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ID.ValueString(),
		contextMap,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Broker context", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerContextResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BrokerContextResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting broker context", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.DeleteBrokerContext(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Failed to delete Broker context", err.Error())
		return
	}
}

func (r *BrokerContextResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import Not Supported",
		"Broker contexts cannot be imported. Please create a new context.",
	)
}
