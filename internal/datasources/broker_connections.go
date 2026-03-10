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
var _ datasource.DataSource = &BrokerConnectionsDataSource{}

// NewBrokerConnectionsDataSource creates a new data source instance
func NewBrokerConnectionsDataSource() datasource.DataSource {
	return &BrokerConnectionsDataSource{}
}

// BrokerConnectionsDataSource defines the data source implementation.
type BrokerConnectionsDataSource struct {
	client *client.Client
}

// BrokerConnectionsDataSourceModel describes the data source data model.
type BrokerConnectionsDataSourceModel struct {
	ID           types.String            `tfsdk:"id"`
	TenantID     types.String            `tfsdk:"tenant_id"`
	InstallID    types.String            `tfsdk:"install_id"`
	DeploymentID types.String            `tfsdk:"deployment_id"`
	Connections  []BrokerConnectionModel `tfsdk:"connections"`
}

// BrokerConnectionModel represents a single connection in the list
type BrokerConnectionModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	DeploymentID  types.String `tfsdk:"deployment_id"`
	Configuration types.Map    `tfsdk:"configuration"`
}

func (d *BrokerConnectionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_connections"
}

func (d *BrokerConnectionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Broker connections for a deployment.",
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
				Description: "The Broker app installation ID.",
				Required:    true,
			},
			"deployment_id": schema.StringAttribute{
				Description: "The deployment ID to list connections for.",
				Required:    true,
			},
			"connections": schema.ListNestedAttribute{
				Description: "The list of Broker connections.",
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

func (d *BrokerConnectionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BrokerConnectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BrokerConnectionsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker connections", map[string]interface{}{
		"deployment_id": data.DeploymentID.ValueString(),
	})

	result, err := d.client.ListBrokerConnections(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Broker connections", err.Error())
		return
	}

	// Set ID for the data source
	data.ID = types.StringValue(data.DeploymentID.ValueString())

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

	tflog.Trace(ctx, "Read broker connections", map[string]interface{}{
		"count": len(connections),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
