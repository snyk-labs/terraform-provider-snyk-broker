// Copyright (c) Snyk Ltd.

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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
var _ resource.Resource = &BrokerAppInstallResource{}
var _ resource.ResourceWithImportState = &BrokerAppInstallResource{}

// NewBrokerAppInstallResource creates a new resource instance
func NewBrokerAppInstallResource() resource.Resource {
	return &BrokerAppInstallResource{}
}

// BrokerAppInstallResource defines the resource implementation.
type BrokerAppInstallResource struct {
	client *client.Client
}

// BrokerAppInstallResourceModel describes the resource data model.
type BrokerAppInstallResourceModel struct {
	ID        types.String   `tfsdk:"id"`
	OrgID     types.String   `tfsdk:"org_id"`
	AppID     types.String   `tfsdk:"app_id"`
	InstallID types.String   `tfsdk:"install_id"`
	ClientID  types.String   `tfsdk:"client_id"`
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
}

func (r *BrokerAppInstallResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_app_install"
}

func (r *BrokerAppInstallResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Installs the Snyk Broker App to an organization. This is a prerequisite for creating broker deployments.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID to install the Broker app to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"app_id": schema.StringAttribute{
				Description: "The Snyk Broker App ID. This is region-specific and can be obtained from Snyk documentation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"install_id": schema.StringAttribute{
				Description: "The resulting installation ID after the app is installed.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_id": schema.StringAttribute{
				Description: "The OAuth client ID for the installed app.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Delete: true,
			}),
		},
	}
}

func (r *BrokerAppInstallResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BrokerAppInstallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BrokerAppInstallResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	tflog.Debug(ctx, "Creating broker app install", map[string]interface{}{
		"org_id": data.OrgID.ValueString(),
		"app_id": data.AppID.ValueString(),
	})

	result, err := r.client.InstallBrokerApp(ctx, data.OrgID.ValueString(), data.AppID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to install Broker app", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.InstallID = types.StringValue(result.Data.ID)
	data.ClientID = types.StringValue(result.Data.Attributes.ClientID)

	tflog.Trace(ctx, "Created broker app install", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerAppInstallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BrokerAppInstallResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := data.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	// The Broker App install API doesn't have a GET endpoint for individual installs
	// We preserve the state as-is since the install is immutable
	tflog.Debug(ctx, "Reading broker app install", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerAppInstallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// App installs are immutable - any changes require replacement
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Broker app installs cannot be updated. Changes require replacement.",
	)
}

func (r *BrokerAppInstallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BrokerAppInstallResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := data.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	tflog.Debug(ctx, "Deleting broker app install", map[string]interface{}{
		"id":     data.ID.ValueString(),
		"org_id": data.OrgID.ValueString(),
	})

	err := r.client.UninstallBrokerApp(ctx, data.OrgID.ValueString(), data.InstallID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to uninstall Broker app", err.Error())
		return
	}

	tflog.Trace(ctx, "Deleted broker app install", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *BrokerAppInstallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: org_id:install_id
	// Since we can't read the full state, we need all components
	resp.Diagnostics.AddError(
		"Import Not Supported",
		"Broker app installs cannot be imported. Please create a new installation.",
	)
}
