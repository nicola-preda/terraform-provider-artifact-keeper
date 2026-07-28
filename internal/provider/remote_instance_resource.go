package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*remoteInstanceResource)(nil)
	_ resource.ResourceWithConfigure   = (*remoteInstanceResource)(nil)
	_ resource.ResourceWithImportState = (*remoteInstanceResource)(nil)
)

func NewRemoteInstanceResource() resource.Resource { return &remoteInstanceResource{} }

type remoteInstanceResource struct {
	client *client.Client
}

type remoteInstanceResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	URL       types.String `tfsdk:"url"`
	APIKey    types.String `tfsdk:"api_key"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *remoteInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote_instance"
}

func (r *remoteInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A registered remote Artifact Keeper instance that this instance can proxy requests to. Owned by the authenticating user. The API has no update endpoint, so changing any field forces a new instance, and the API key is not returned.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Remote instance UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name for the remote instance. Changing this forces a new instance.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Base URL of the remote instance (e.g. `https://ak.example.com`). Changing this forces a new instance.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"api_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "API key (bearer token) used to authenticate against the remote instance. Not returned by the API. Changing this forces a new instance.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *remoteInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *remoteInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan remoteInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inst, err := r.client.CreateRemoteInstance(ctx, client.CreateRemoteInstanceRequest{
		Name:   plan.Name.ValueString(),
		URL:    plan.URL.ValueString(),
		APIKey: plan.APIKey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating remote instance", err.Error())
		return
	}

	state := remoteInstanceToModel(inst)
	state.APIKey = plan.APIKey
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *remoteInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state remoteInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inst, err := r.client.GetRemoteInstance(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading remote instance", err.Error())
		return
	}

	refreshed := remoteInstanceToModel(inst)
	refreshed.APIKey = state.APIKey
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update is unreachable: every configurable attribute forces replacement.
func (r *remoteInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan remoteInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *remoteInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state remoteInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRemoteInstance(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting remote instance", err.Error())
	}
}

func (r *remoteInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func remoteInstanceToModel(i *client.RemoteInstance) remoteInstanceResourceModel {
	return remoteInstanceResourceModel{
		ID:        types.StringValue(i.ID),
		Name:      types.StringValue(i.Name),
		URL:       types.StringValue(i.URL),
		CreatedAt: types.StringValue(i.CreatedAt),
	}
}
