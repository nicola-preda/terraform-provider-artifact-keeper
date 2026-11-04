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

// serviceAccountResource manages a service account through /service-accounts,
// except for is_admin: the service-account endpoints neither accept nor return
// that flag, but a service account is a row in users, so it is set with
// PATCH /users/{id} and read with GET /users/{id}, like artifactkeeper_user.
type serviceAccountResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Username    types.String `tfsdk:"username"`
	DisplayName types.String `tfsdk:"display_name"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	IsAdmin     types.Bool   `tfsdk:"is_admin"`
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
			"is_admin": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the service account is an Artifact Keeper administrator. Defaults to `false`. The service-account endpoints do not carry this flag, so it is set with `PATCH /users/{id}` and read with `GET /users/{id}`, which accept a service-account id. A token of an administrator service account acts as an administrator only when it was minted with the `admin` (or `*`) scope; see `artifactkeeper_service_account_token`.",
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

	// Service accounts are created as non-administrators and the create request
	// cannot change that, so promote through the user endpoint when asked.
	state.IsAdmin = types.BoolValue(false)
	if !plan.IsAdmin.IsNull() && !plan.IsAdmin.IsUnknown() && plan.IsAdmin.ValueBool() {
		isAdmin, updatedAt, err := r.setAdmin(ctx, sa.ID, plan.IsAdmin.ValueBoolPointer())
		if err != nil {
			resp.Diagnostics.AddError("Error setting service account administrator flag", err.Error())
			return
		}
		state.IsAdmin = isAdmin
		state.UpdatedAt = updatedAt
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// setAdmin sets is_admin through PATCH /users/{id} and returns the flag and
// the service account's updated_at, which that write bumps.
func (r *serviceAccountResource) setAdmin(ctx context.Context, id string, isAdmin *bool) (types.Bool, types.String, error) {
	u, err := r.client.UpdateUser(ctx, id, client.UpdateUserRequest{IsAdmin: isAdmin})
	if err != nil {
		return types.BoolNull(), types.StringNull(), err
	}
	sa, err := r.client.GetServiceAccount(ctx, id)
	if err != nil {
		return types.BoolNull(), types.StringNull(), err
	}
	return types.BoolValue(u.IsAdmin), types.StringValue(sa.UpdatedAt), nil
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

	// The service-account response has no is_admin; the user row does. The
	// account exists (the call above succeeded), so a 404 here is an error.
	u, err := r.client.GetUser(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading service account administrator flag", err.Error())
		return
	}
	refreshed.IsAdmin = types.BoolValue(u.IsAdmin)

	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
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

	// is_admin goes through the user endpoint. Every PATCH there is audited,
	// so only send it when the flag changes.
	state.IsAdmin = prior.IsAdmin
	if !plan.IsAdmin.IsNull() && !plan.IsAdmin.IsUnknown() && !plan.IsAdmin.Equal(prior.IsAdmin) {
		isAdmin, updatedAt, err := r.setAdmin(ctx, plan.ID.ValueString(), plan.IsAdmin.ValueBoolPointer())
		if err != nil {
			resp.Diagnostics.AddError("Error setting service account administrator flag", err.Error())
			return
		}
		state.IsAdmin = isAdmin
		state.UpdatedAt = updatedAt
	}

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
