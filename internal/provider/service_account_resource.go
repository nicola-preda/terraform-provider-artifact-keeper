package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*serviceAccountResource)(nil)
	_ resource.ResourceWithConfigure   = (*serviceAccountResource)(nil)
	_ resource.ResourceWithImportState = (*serviceAccountResource)(nil)
)

func NewServiceAccountResource() resource.Resource { return &serviceAccountResource{} }

type serviceAccountResource struct {
	client *client.Client
}

type serviceAccountResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Username    types.String `tfsdk:"username"`
	DisplayName types.String `tfsdk:"display_name"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *serviceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *serviceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A service account: a machine identity that owns API tokens independently of any human user. Tokens are managed separately. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Service account UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name for the service account. The server derives the immutable `username` from it (lower-cased, spaces to hyphens, prefixed with `svc-`). Not returned by the API, so it is preserved from configuration. Changing it forces a new service account.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-derived username (e.g. `svc-ci-runner`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"display_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Human-readable display name (sent as `description` on create).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the service account is active. Defaults to `true`. Set to `false` to disable it (its API tokens stop authenticating).",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 last-update timestamp.",
			},
		},
	}
}

func (r *serviceAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateServiceAccountRequest{Name: plan.Name.ValueString()}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		createReq.Description = plan.DisplayName.ValueStringPointer()
	}

	sa, err := r.client.CreateServiceAccount(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service account", err.Error())
		return
	}

	// is_active is not part of the create request (accounts are created active),
	// so reconcile it with a follow-up update when the caller asked otherwise.
	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() && plan.IsActive.ValueBool() != sa.IsActive {
		sa, err = r.client.UpdateServiceAccount(ctx, sa.ID, client.UpdateServiceAccountRequest{IsActive: plan.IsActive.ValueBoolPointer()})
		if err != nil {
			resp.Diagnostics.AddError("Error setting service account active state", err.Error())
			return
		}
	}

	state := serviceAccountToModel(sa)
	state.Name = plan.Name // not returned by the API; preserved from config
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sa, err := r.client.GetServiceAccount(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service account", err.Error())
		return
	}

	refreshed := serviceAccountToModel(sa)
	refreshed.Name = state.Name // not returned by the API; preserved from config
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateServiceAccountRequest{}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		updateReq.DisplayName = plan.DisplayName.ValueStringPointer()
	}
	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() {
		updateReq.IsActive = plan.IsActive.ValueBoolPointer()
	}

	sa, err := r.client.UpdateServiceAccount(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating service account", err.Error())
		return
	}

	state := serviceAccountToModel(sa)
	state.Name = plan.Name // not returned by the API; preserved from config
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteServiceAccount(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting service account", err.Error())
	}
}

func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func serviceAccountToModel(sa *client.ServiceAccount) serviceAccountResourceModel {
	return serviceAccountResourceModel{
		ID:          types.StringValue(sa.ID),
		Username:    types.StringValue(sa.Username),
		DisplayName: stringPointerValue(sa.DisplayName),
		IsActive:    types.BoolValue(sa.IsActive),
		CreatedAt:   types.StringValue(sa.CreatedAt),
		UpdatedAt:   types.StringValue(sa.UpdatedAt),
	}
}
