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
	_ resource.Resource                = (*userResource)(nil)
	_ resource.ResourceWithConfigure   = (*userResource)(nil)
	_ resource.ResourceWithImportState = (*userResource)(nil)
)

func NewUserResource() resource.Resource { return &userResource{} }

type userResource struct {
	client *client.Client
}

type userResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Username           types.String `tfsdk:"username"`
	Email              types.String `tfsdk:"email"`
	Password           types.String `tfsdk:"password"`
	DisplayName        types.String `tfsdk:"display_name"`
	IsAdmin            types.Bool   `tfsdk:"is_admin"`
	IsActive           types.Bool   `tfsdk:"is_active"`
	AuthProvider       types.String `tfsdk:"auth_provider"`
	MustChangePassword types.Bool   `tfsdk:"must_change_password"`
	LastLoginAt        types.String `tfsdk:"last_login_at"`
	GeneratedPassword  types.String `tfsdk:"generated_password"`
	CreatedAt          types.String `tfsdk:"created_at"`
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a local user. SSO users are provisioned on login and need not be managed here.",
		Attributes: map[string]schema.Attribute{
			"id":                   schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"username":             schema.StringAttribute{Required: true, MarkdownDescription: "Unique username. Changing it forces a new user.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"email":                schema.StringAttribute{Required: true},
			"password":             schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Initial password; if omitted a password is generated (see `generated_password`). Not returned by the API. Changing it forces a new user.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"display_name":         schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"is_admin":             schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: "Defaults to `false`.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
			"is_active":            schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: "Defaults to `true`.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
			"auth_provider":        schema.StringAttribute{Computed: true},
			"must_change_password": schema.BoolAttribute{Computed: true},
			"last_login_at":        schema.StringAttribute{Computed: true},
			"generated_password":   schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Auto-generated password, set only when `password` was omitted at creation.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"created_at":           schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createReq := client.CreateUserRequest{
		Username:    plan.Username.ValueString(),
		Email:       plan.Email.ValueString(),
		Password:    optionalString(plan.Password),
		DisplayName: optionalString(plan.DisplayName),
	}
	if !plan.IsAdmin.IsNull() && !plan.IsAdmin.IsUnknown() {
		createReq.IsAdmin = plan.IsAdmin.ValueBoolPointer()
	}
	out, err := r.client.CreateUser(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}
	created := out.User

	// The create endpoint has no is_active field; the server always creates a
	// user active. If the config asked for is_active = false, follow up with a
	// PATCH so the final state matches, instead of returning an inconsistent result.
	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() && !plan.IsActive.ValueBool() {
		active := false
		updated, err := r.client.UpdateUser(ctx, created.ID, client.UpdateUserRequest{IsActive: &active})
		if err != nil {
			resp.Diagnostics.AddError("Error deactivating user after create", err.Error())
			return
		}
		created = *updated
	}

	state := userToModel(&created)
	state.Password = plan.Password
	state.GeneratedPassword = stringPointerValue(out.GeneratedPassword)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	u, err := r.client.GetUser(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}
	refreshed := userToModel(u)
	refreshed.Password = state.Password
	refreshed.GeneratedPassword = state.GeneratedPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq := client.UpdateUserRequest{Email: plan.Email.ValueStringPointer()}
	updateReq.DisplayName = optionalString(plan.DisplayName)
	if !plan.IsAdmin.IsNull() && !plan.IsAdmin.IsUnknown() {
		updateReq.IsAdmin = plan.IsAdmin.ValueBoolPointer()
	}
	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() {
		updateReq.IsActive = plan.IsActive.ValueBoolPointer()
	}
	u, err := r.client.UpdateUser(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}
	refreshed := userToModel(u)
	refreshed.Password = state.Password
	refreshed.GeneratedPassword = state.GeneratedPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteUser(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func userToModel(u *client.User) userResourceModel {
	return userResourceModel{
		ID:                 types.StringValue(u.ID),
		Username:           types.StringValue(u.Username),
		Email:              types.StringValue(u.Email),
		DisplayName:        stringPointerValue(u.DisplayName),
		IsAdmin:            types.BoolValue(u.IsAdmin),
		IsActive:           types.BoolValue(u.IsActive),
		AuthProvider:       types.StringValue(u.AuthProvider),
		MustChangePassword: types.BoolValue(u.MustChangePassword),
		LastLoginAt:        stringPointerValue(u.LastLoginAt),
		CreatedAt:          types.StringValue(u.CreatedAt),
	}
}
