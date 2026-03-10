// Copyright (c) Snyk Ltd.

package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/snyk-labs/snyk-broker-provider/internal/client"
	"github.com/snyk-labs/snyk-broker-provider/internal/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BrokerConnectionsForOrgDataSource{}

// NewBrokerConnectionsForOrgDataSource creates a new data source instance
func NewBrokerConnectionsForOrgDataSource() datasource.DataSource {
	return &BrokerConnectionsForOrgDataSource{}
}

// BrokerConnectionsForOrgDataSource defines the data source implementation.
type BrokerConnectionsForOrgDataSource struct {
	client *client.Client
}

// BrokerConnectionsForOrgDataSourceModel describes the data source data model.
type BrokerConnectionsForOrgDataSourceModel struct {
	ID          types.String            `tfsdk:"id"`
	OrgID       types.String            `tfsdk:"org_id"`
	Connections []BrokerConnectionModel `tfsdk:"connections"`
}

func (d *BrokerConnectionsForOrgDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_connections_for_org"
}

func (d *BrokerConnectionsForOrgDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Broker connections for a given organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The data source identifier.",
				Computed:    true,
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID to list connections for.",
				Required:    true,
			},
			"connections": schema.ListNestedAttribute{
				Description: "The list of Broker connections for the organization.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The connection ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The connection name.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The connection type.",
							Computed:    true,
						},
						"deployment_id": schema.StringAttribute{
							Description: "The deployment ID this connection belongs to.",
							Computed:    true,
						},
						"configuration": schema.MapAttribute{
							Description: "The connection configuration.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *BrokerConnectionsForOrgDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BrokerConnectionsForOrgDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BrokerConnectionsForOrgDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker connections for org", map[string]interface{}{
		"org_id": data.OrgID.ValueString(),
	})

	result, err := d.client.ListBrokerConnectionsForOrg(ctx, data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Broker connections for organization", err.Error())
		return
	}

	// Set ID for the data source
	data.ID = types.StringValue(data.OrgID.ValueString())

	// Convert response to model
	connections := make([]BrokerConnectionModel, 0, len(result.Data))
	for _, conn := range result.Data {
		connection := BrokerConnectionModel{
			ID:           types.StringValue(conn.ID),
			Name:         types.StringValue(conn.Attributes.Name),
			Type:         types.StringValue(conn.Attributes.Configuration.Type),
			DeploymentID: types.StringValue(conn.Attributes.DeploymentID),
		}

		// Convert configuration
		configMap := make(map[string]string)
		for k, v := range conn.Attributes.Configuration.Required {
			if strVal, ok := v.(string); ok {
				configMap[k] = strVal
			}
		}
		configValue, diags := types.MapValueFrom(ctx, types.StringType, configMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		connection.Configuration = configValue

		connections = append(connections, connection)
	}

	data.Connections = connections

	tflog.Trace(ctx, "Read broker connections for org", map[string]interface{}{
		"count": len(connections),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
