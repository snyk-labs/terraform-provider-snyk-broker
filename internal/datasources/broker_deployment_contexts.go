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
var _ datasource.DataSource = &BrokerDeploymentContextsDataSource{}

// NewBrokerDeploymentContextsDataSource creates a new data source instance
func NewBrokerDeploymentContextsDataSource() datasource.DataSource {
	return &BrokerDeploymentContextsDataSource{}
}

// BrokerDeploymentContextsDataSource defines the data source implementation.
type BrokerDeploymentContextsDataSource struct {
	client *client.Client
}

// BrokerDeploymentContextsDataSourceModel describes the data source data model.
type BrokerDeploymentContextsDataSourceModel struct {
	ID           types.String                    `tfsdk:"id"`
	TenantID     types.String                    `tfsdk:"tenant_id"`
	InstallID    types.String                    `tfsdk:"install_id"`
	DeploymentID types.String                    `tfsdk:"deployment_id"`
	Contexts     []BrokerDeploymentContextModel  `tfsdk:"contexts"`
}

// BrokerDeploymentContextModel represents a single context in the list
type BrokerDeploymentContextModel struct {
	ID             types.String   `tfsdk:"id"`
	Context        types.Map      `tfsdk:"context"`
	ConnectionIDs  []types.String `tfsdk:"connection_ids"`
	IntegrationIDs []types.String `tfsdk:"integration_ids"`
}

func (d *BrokerDeploymentContextsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_deployment_contexts"
}

func (d *BrokerDeploymentContextsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Broker contexts for a deployment.",
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
				Description: "The deployment ID to list contexts for.",
				Required:    true,
			},
			"contexts": schema.ListNestedAttribute{
				Description: "The list of Broker contexts.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The context ID.",
							Computed:    true,
						},
						"context": schema.MapAttribute{
							Description: "The context configuration as key-value pairs.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"connection_ids": schema.ListAttribute{
							Description: "The IDs of connections associated with this context.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"integration_ids": schema.ListAttribute{
							Description: "The IDs of integrations associated with this context.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *BrokerDeploymentContextsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BrokerDeploymentContextsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BrokerDeploymentContextsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker deployment contexts", map[string]interface{}{
		"deployment_id": data.DeploymentID.ValueString(),
	})

	result, err := d.client.ListDeploymentContexts(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Broker deployment contexts", err.Error())
		return
	}

	data.ID = types.StringValue(data.DeploymentID.ValueString())

	contexts := make([]BrokerDeploymentContextModel, 0, len(result.Data))
	for _, brokerCtx := range result.Data {
		contextModel := BrokerDeploymentContextModel{
			ID: types.StringValue(brokerCtx.ID),
		}

		contextValue, diags := types.MapValueFrom(ctx, types.StringType, brokerCtx.Attributes.Context)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		contextModel.Context = contextValue

		connectionIDs := []types.String{}
		integrationIDs := []types.String{}

		if brokerCtx.Relationships != nil {
			for _, conn := range brokerCtx.Relationships.BrokerConnections {
				connectionIDs = append(connectionIDs, types.StringValue(conn.Data.ID))
			}
			for _, integration := range brokerCtx.Relationships.AppliedIntegrations {
				integrationIDs = append(integrationIDs, types.StringValue(integration.Data.ID))
			}
		}

		contextModel.ConnectionIDs = connectionIDs
		contextModel.IntegrationIDs = integrationIDs

		contexts = append(contexts, contextModel)
	}

	data.Contexts = contexts

	tflog.Trace(ctx, "Read broker deployment contexts", map[string]interface{}{
		"count": len(contexts),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
