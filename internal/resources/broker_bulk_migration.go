// Copyright (c) Snyk Ltd.

package resources

import (
	"context"
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
var _ resource.Resource = &BrokerBulkMigrationResource{}

// NewBrokerBulkMigrationResource creates a new resource instance
func NewBrokerBulkMigrationResource() resource.Resource {
	return &BrokerBulkMigrationResource{}
}

// BrokerBulkMigrationResource defines the resource implementation.
type BrokerBulkMigrationResource struct {
	client *client.Client
}

// BrokerBulkMigrationResourceModel describes the resource data model.
type BrokerBulkMigrationResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TenantID     types.String `tfsdk:"tenant_id"`
	InstallID    types.String `tfsdk:"install_id"`
	DeploymentID types.String `tfsdk:"deployment_id"`
	ConnectionID types.String `tfsdk:"connection_id"`
	OrgIDs       types.List   `tfsdk:"org_ids"`
	Status       types.String `tfsdk:"status"`
}

func (r *BrokerBulkMigrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_bulk_migration"
}

func (r *BrokerBulkMigrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Performs bulk migration of organization integrations to Universal Broker. This migrates multiple organizations at once to use a Broker connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this migration.",
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
				Description: "The Broker app install ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Description: "The Broker deployment ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connection_id": schema.StringAttribute{
				Description: "The Broker connection ID to migrate organizations to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_ids": schema.ListAttribute{
				Description: "The list of organization IDs to migrate to the Broker connection.",
				Required:    true,
				ElementType: types.StringType,
			},
			"status": schema.StringAttribute{
				Description: "The status of the migration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *BrokerBulkMigrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BrokerBulkMigrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BrokerBulkMigrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert org_ids list to []string
	var orgIDs []string
	resp.Diagnostics.Append(data.OrgIDs.ElementsAs(ctx, &orgIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating broker bulk migration", map[string]interface{}{
		"connection_id": data.ConnectionID.ValueString(),
		"install_id":    data.InstallID.ValueString(),
		"deployment_id": data.DeploymentID.ValueString(),
		"org_count":     len(orgIDs),
	})

	result, err := r.client.CreateBrokerBulkMigration(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		data.ConnectionID.ValueString(),
		orgIDs,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Broker bulk migration", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.Status = types.StringValue(result.Data.Attributes.Status)

	tflog.Trace(ctx, "Created broker bulk migration", map[string]interface{}{
		"id":     data.ID.ValueString(),
		"status": data.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerBulkMigrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BrokerBulkMigrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker bulk migration", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	// Bulk migrations are fire-and-forget, so we just preserve state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerBulkMigrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BrokerBulkMigrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert org_ids list to []string
	var orgIDs []string
	resp.Diagnostics.Append(data.OrgIDs.ElementsAs(ctx, &orgIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating broker bulk migration (triggering new migration)", map[string]interface{}{
		"connection_id": data.ConnectionID.ValueString(),
		"install_id":    data.InstallID.ValueString(),
		"deployment_id": data.DeploymentID.ValueString(),
		"org_count":     len(orgIDs),
	})

	// Trigger a new migration for the updated list of orgs
	result, err := r.client.CreateBrokerBulkMigration(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		data.ConnectionID.ValueString(),
		orgIDs,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Broker bulk migration", err.Error())
		return
	}

	data.Status = types.StringValue(result.Data.Attributes.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerBulkMigrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BrokerBulkMigrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting broker bulk migration (removing from state only)", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	// Bulk migrations cannot be undone - we just remove from state
	// The organizations remain migrated
}
