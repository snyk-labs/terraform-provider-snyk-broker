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
var _ resource.Resource = &BrokerCredentialResource{}
var _ resource.ResourceWithImportState = &BrokerCredentialResource{}

// NewBrokerCredentialResource creates a new resource instance
func NewBrokerCredentialResource() resource.Resource {
	return &BrokerCredentialResource{}
}

// BrokerCredentialResource defines the resource implementation.
type BrokerCredentialResource struct {
	client *client.Client
}

// BrokerCredentialResourceModel describes the resource data model.
type BrokerCredentialResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	TenantID                types.String `tfsdk:"tenant_id"`
	InstallID               types.String `tfsdk:"install_id"`
	DeploymentID            types.String `tfsdk:"deployment_id"`
	EnvironmentVariableName types.String `tfsdk:"environment_variable_name"`
	Type                    types.String `tfsdk:"type"`
	Comment                 types.String `tfsdk:"comment"`
}

func (r *BrokerCredentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_broker_credential"
}

func (r *BrokerCredentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Snyk Broker credential reference. Credential references define the environment variables that the Broker client expects to contain secrets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this credential reference.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID for the credential.",
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
				Description: "The deployment ID this credential belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_variable_name": schema.StringAttribute{
				Description: "The name of the environment variable that will contain the secret (e.g., MY_GITHUB_TOKEN).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of credential (e.g., github, gitlab, bitbucket-server, jira, artifactory, nexus, azure-repos, container-registry-agent).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"comment": schema.StringAttribute{
				Description: "A description of this credential for documentation purposes.",
				Optional:    true,
			},
		},
	}
}

func (r *BrokerCredentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BrokerCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BrokerCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cred := client.BrokerCredentialAttributes{
		EnvironmentVariableName: data.EnvironmentVariableName.ValueString(),
		Type:                    data.Type.ValueString(),
	}
	if !data.Comment.IsNull() {
		cred.Comment = data.Comment.ValueString()
	}

	tflog.Debug(ctx, "Creating broker credential", map[string]interface{}{
		"deployment_id": data.DeploymentID.ValueString(),
		"env_var_name":  data.EnvironmentVariableName.ValueString(),
		"type":          data.Type.ValueString(),
	})

	result, err := r.client.CreateBrokerCredential(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		cred,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Broker credential", err.Error())
		return
	}

	if len(result.Data) == 0 {
		resp.Diagnostics.AddError("Failed to create Broker credential", "API returned empty response")
		return
	}

	data.ID = types.StringValue(result.Data[0].ID)

	tflog.Trace(ctx, "Created broker credential", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BrokerCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading broker credential", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	// List all credentials and find the one matching our ID
	result, err := r.client.ListBrokerCredentials(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Broker credentials", err.Error())
		return
	}

	// Find the credential with matching ID
	found := false
	for _, cred := range result.Data {
		if cred.ID == data.ID.ValueString() {
			found = true
			data.EnvironmentVariableName = types.StringValue(cred.Attributes.EnvironmentVariableName)
			data.Type = types.StringValue(cred.Attributes.Type)
			if cred.Attributes.Comment != "" {
				data.Comment = types.StringValue(cred.Attributes.Comment)
			}
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Credentials are mostly immutable - only comment can be updated
	// For now, we require replacement for all changes
	var data BrokerCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Just update state since comment changes don't require API call
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrokerCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BrokerCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting broker credential", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.DeleteBrokerCredential(
		ctx,
		data.TenantID.ValueString(),
		data.InstallID.ValueString(),
		data.DeploymentID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		// Check if it's a 404 - already deleted (use errors.As for wrapped errors)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Failed to delete Broker credential", err.Error())
		return
	}
}

func (r *BrokerCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import Not Supported",
		"Broker credentials cannot be imported. Please create a new credential reference.",
	)
}
