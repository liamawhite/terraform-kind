package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ClusterResource{}
var _ resource.ResourceWithImportState = &ClusterResource{}

func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

// ClusterResource defines the resource implementation.
type ClusterResource struct{}

// ClusterResourceModel describes the resource data model.
type ClusterResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	NodeVersion types.String `tfsdk:"node_version"`
	Kubeconfig  types.String `tfsdk:"kubeconfig"`
}

func (r *ClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *ClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Cluster resource is a Kubernetes in Docker (KIND) cluster.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the cluster",
				Required:            true,
			},
			"node_version": schema.StringAttribute{
				MarkdownDescription: "Version of the cluster to be created, must be one of the tags available [here](https://hub.docker.com/r/kindest/node/tags).",
				Required:            true,
			},
			"kubeconfig": schema.StringAttribute{
				MarkdownDescription: "The kubeconfig for interacting with the cluster.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Not sure what to actually do here as none of the attributes are updateable so far...
func (r *ClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// // Prevent panic if the provider has not been configured.
	// if req.ProviderData == nil {
	// 	return
	// }

	// client, ok := req.ProviderData.(*http.Client)

	// if !ok {
	// 	resp.Diagnostics.AddError(
	// 		"Unexpected Resource Configure Type",
	// 		fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
	// 	)

	// 	return
	// }

	// r.client = client
}

func (r *ClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Do the create...
	createCluster(data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ClusterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse id to get name and node version
	split := strings.Split(data.Id.ValueString(), "/")
	if len(split) != 2 {
		resp.Diagnostics.AddError("failed to parse cluster id", data.Id.ValueString())
		return
	}
	data.Name = types.StringValue(split[0])
	data.NodeVersion = types.StringValue(split[1])

	// Retrieve kubeconfig using kind
	provider := cluster.NewProvider(cluster.ProviderWithLogger(cmd.NewLogger()))
	kubeconfig, err := provider.KubeConfig(data.Name.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError("failed to retrieve cluster kubeconfig", err.Error())
		return
	}
	data.Kubeconfig = types.StringValue(kubeconfig)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ClusterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Do the delete
	deleteCluster(data, &resp.Diagnostics)
}

func (r *ClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createCluster(data *ClusterResourceModel, diags *diag.Diagnostics) {
	opts := []cluster.CreateOption{
		cluster.CreateWithWaitForReady(defaultCreateTimeout),
		cluster.CreateWithNodeImage(fmt.Sprintf("kindest/node:%s", data.NodeVersion.ValueString())),
	}
	provider := cluster.NewProvider(cluster.ProviderWithLogger(cmd.NewLogger()))
	err := provider.Create(data.Name.ValueString(), opts...)
	if err != nil {
		diags.AddError("failed to create cluster", err.Error())
		return
	}

	// Set the computed variables
	data.Id = types.StringValue(fmt.Sprintf("%s/%s", data.Name.ValueString(), data.NodeVersion.ValueString()))
	kubeconfig, err := provider.KubeConfig(data.Name.ValueString(), false)
	if err != nil {
		diags.AddError("failed to retrieve cluster kubeconfig", err.Error())
		return
	}
	data.Kubeconfig = types.StringValue(kubeconfig)
}

func deleteCluster(data *ClusterResourceModel, diags *diag.Diagnostics) {
	provider := cluster.NewProvider(cluster.ProviderWithLogger(cmd.NewLogger()))
	err := provider.Delete(data.Name.ValueString(), "")
	if err != nil {
		diags.AddError("failed to delete cluster", err.Error())
		return
	}
}
