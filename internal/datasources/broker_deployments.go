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
var _ datasource.DataSource = &BrokerDeploymentsDataSource{}

// NewBrokerDeploymentsDataSource creates a new data source instance
func NewBrokerDeploymentsDataSource() datasource.DataSource {
	return &BrokerDeploymentsDataSource{}
}

// BrokerDeploymentsDataSource defines the data source implementation.
type BrokerDeploymentsDataSource struct {
	client *client.Client
}

// BrokerDeploymentsDataSourceModel describes the data source data model.
type BrokerDeploymentsDataSourceModel struct {
	ID          types.String            `tfsdk:"id"`
	TenantID    types.String            `tfsdk:"tenant_id"`
	InstallID   types.String            `tfsdk:"install_id"`
	Deployments []BrokerDeploymentModel `tfsdk:"deployments"`
}

// BrokerDeploymentModel represents a single deployment in the list
type BrokerDeploymentModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Metadata types.Map    `tfsdk:"metadata"`
}

func (d *BrokerDeploymentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_deployments"
}

func (d *BrokerDeploymentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Broker deployments for a tenant.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The data source identifier.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID to list deployments for.",
				Required:    true,
			},
			"install_id": schema.StringAttribute{
				Description: "The Broker app installation ID. If not provided, lists all deployments for the tenant.",
				Optional:    true,
			},
			"deployments": schema.ListNestedAttribute{
				Description: "The list of Broker deployments.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The deployment ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The deployment name.",
							Computed:    true,
						},
						"metadata": schema.MapAttribute{
							Description: "The deployment metadata.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *BrokerDeploymentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BrokerDeploymentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BrokerDeploymentsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker deployments", map[string]interface{}{
		"tenant_id": data.TenantID.ValueString(),
	})

	var result *client.BrokerDeploymentsListResponse
	var err error

	if !data.InstallID.IsNull() && !data.InstallID.IsUnknown() {
		result, err = d.client.ListBrokerDeployments(ctx, data.TenantID.ValueString(), data.InstallID.ValueString())
	} else {
		result, err = d.client.ListBrokerDeploymentsForTenant(ctx, data.TenantID.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError("Failed to list Broker deployments", err.Error())
		return
	}

	// Set ID for the data source
	data.ID = types.StringValue(data.TenantID.ValueString())

	// Convert response to model
	deployments := make([]BrokerDeploymentModel, 0, len(result.Data))
	for _, dep := range result.Data {
		deployment := BrokerDeploymentModel{
			ID: types.StringValue(dep.ID),
		}

		// Extract name from metadata
		if name, ok := dep.Attributes.Metadata["deployment_name"].(string); ok {
			deployment.Name = types.StringValue(name)
		} else {
			deployment.Name = types.StringNull()
		}

		// Convert metadata
		metadataMap := make(map[string]string)
		for k, v := range dep.Attributes.Metadata {
			if strVal, ok := v.(string); ok {
				metadataMap[k] = strVal
			}
		}
		metadataValue, diags := types.MapValueFrom(ctx, types.StringType, metadataMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		deployment.Metadata = metadataValue

		deployments = append(deployments, deployment)
	}

	data.Deployments = deployments

	tflog.Trace(ctx, "Read broker deployments", map[string]interface{}{
		"count": len(deployments),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
