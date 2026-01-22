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
var _ datasource.DataSource = &BrokerConnectionIntegrationsDataSource{}

// NewBrokerConnectionIntegrationsDataSource creates a new data source instance
func NewBrokerConnectionIntegrationsDataSource() datasource.DataSource {
	return &BrokerConnectionIntegrationsDataSource{}
}

// BrokerConnectionIntegrationsDataSource defines the data source implementation.
type BrokerConnectionIntegrationsDataSource struct {
	client *client.Client
}

// BrokerConnectionIntegrationsDataSourceModel describes the data source data model.
type BrokerConnectionIntegrationsDataSourceModel struct {
	ID           types.String                       `tfsdk:"id"`
	TenantID     types.String                       `tfsdk:"tenant_id"`
	ConnectionID types.String                       `tfsdk:"connection_id"`
	Integrations []BrokerConnectionIntegrationModel `tfsdk:"integrations"`
}

// BrokerConnectionIntegrationModel represents a single integration in the list
type BrokerConnectionIntegrationModel struct {
	ID            types.String `tfsdk:"id"`
	OrgID         types.String `tfsdk:"org_id"`
	IntegrationID types.String `tfsdk:"integration_id"`
	Type          types.String `tfsdk:"type"`
}

func (d *BrokerConnectionIntegrationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_connection_integrations"
}

func (d *BrokerConnectionIntegrationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all integrations using a Broker connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The data source identifier.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID.",
				Required:    true,
			},
			"connection_id": schema.StringAttribute{
				Description: "The Broker connection ID to list integrations for.",
				Required:    true,
			},
			"integrations": schema.ListNestedAttribute{
				Description: "The list of integrations using this Broker connection.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The integration link ID.",
							Computed:    true,
						},
						"org_id": schema.StringAttribute{
							Description: "The organization ID.",
							Computed:    true,
						},
						"integration_id": schema.StringAttribute{
							Description: "The integration ID.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The integration type.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *BrokerConnectionIntegrationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BrokerConnectionIntegrationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BrokerConnectionIntegrationsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker connection integrations", map[string]interface{}{
		"connection_id": data.ConnectionID.ValueString(),
	})

	result, err := d.client.GetBrokerConnectionIntegrations(
		ctx,
		data.TenantID.ValueString(),
		data.ConnectionID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Broker connection integrations", err.Error())
		return
	}

	// Set ID for the data source
	data.ID = types.StringValue(data.ConnectionID.ValueString())

	// Convert response to model
	integrations := make([]BrokerConnectionIntegrationModel, 0, len(result.Data))
	for _, integ := range result.Data {
		integration := BrokerConnectionIntegrationModel{
			ID:            types.StringValue(integ.ID),
			OrgID:         types.StringValue(integ.Attributes.OrgID),
			IntegrationID: types.StringValue(integ.Attributes.IntegrationID),
			Type:          types.StringValue(integ.Attributes.Type),
		}
		integrations = append(integrations, integration)
	}

	data.Integrations = integrations

	tflog.Trace(ctx, "Read broker connection integrations", map[string]interface{}{
		"count": len(integrations),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
