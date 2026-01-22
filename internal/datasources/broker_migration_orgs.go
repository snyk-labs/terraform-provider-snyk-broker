// Copyright (c) Snyk Ltd.

package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
	"github.com/snyk/terraform-provider-snyk-broker/internal/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BrokerMigrationOrgsDataSource{}

// NewBrokerMigrationOrgsDataSource creates a new data source instance
func NewBrokerMigrationOrgsDataSource() datasource.DataSource {
	return &BrokerMigrationOrgsDataSource{}
}

// BrokerMigrationOrgsDataSource defines the data source implementation.
type BrokerMigrationOrgsDataSource struct {
	client *client.Client
}

// BrokerMigrationOrgsDataSourceModel describes the data source data model.
type BrokerMigrationOrgsDataSourceModel struct {
	ID            types.String              `tfsdk:"id"`
	TenantID      types.String              `tfsdk:"tenant_id"`
	InstallID     types.String              `tfsdk:"install_id"`
	DeploymentID  types.String              `tfsdk:"deployment_id"`
	ConnectionID  types.String              `tfsdk:"connection_id"`
	Organizations []BrokerMigrationOrgModel `tfsdk:"organizations"`
}

// BrokerMigrationOrgModel represents a single organization available for migration
type BrokerMigrationOrgModel struct {
	ID      types.String `tfsdk:"id"`
	OrgID   types.String `tfsdk:"org_id"`
	OrgName types.String `tfsdk:"org_name"`
}

func (d *BrokerMigrationOrgsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_migration_orgs"
}

func (d *BrokerMigrationOrgsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists organizations available for bulk migration to Universal Broker.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The data source identifier.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID.",
				Required:    true,
			},
			"install_id": schema.StringAttribute{
				Description: "The Broker app install ID.",
				Required:    true,
			},
			"deployment_id": schema.StringAttribute{
				Description: "The Broker deployment ID.",
				Required:    true,
			},
			"connection_id": schema.StringAttribute{
				Description: "The Broker connection ID to check migration eligibility for.",
				Required:    true,
			},
			"organizations": schema.ListNestedAttribute{
				Description: "The list of organizations available for migration.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The organization record ID.",
							Computed:    true,
						},
						"org_id": schema.StringAttribute{
							Description: "The organization ID.",
							Computed:    true,
						},
						"org_name": schema.StringAttribute{
							Description: "The organization name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *BrokerMigrationOrgsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*common.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = providerData.Client
}

func (d *BrokerMigrationOrgsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BrokerMigrationOrgsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker migration orgs", map[string]interface{}{
		"connection_id": data.ConnectionID.ValueString(),
		"install_id":    data.InstallID.ValueString(),
		"deployment_id": data.DeploymentID.ValueString(),
	})

	result, err := d.client.ListBrokerMigrationOrgs(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		data.ConnectionID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list organizations for migration", err.Error())
		return
	}

	// Set ID for the data source
	data.ID = types.StringValue(data.ConnectionID.ValueString())

	// Convert response to model
	organizations := make([]BrokerMigrationOrgModel, 0, len(result.Data))
	for _, org := range result.Data {
		organization := BrokerMigrationOrgModel{
			ID:      types.StringValue(org.ID),
			OrgID:   types.StringValue(org.Attributes.OrgID),
			OrgName: types.StringValue(org.Attributes.OrgName),
		}
		organizations = append(organizations, organization)
	}

	data.Organizations = organizations

	tflog.Trace(ctx, "Read broker migration orgs", map[string]interface{}{
		"count": len(organizations),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
