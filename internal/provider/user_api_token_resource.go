package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*userApiTokenResource)(nil)
	_ resource.ResourceWithConfigure   = (*userApiTokenResource)(nil)
	_ resource.ResourceWithImportState = (*userApiTokenResource)(nil)
)

func NewUserApiTokenResource() resource.Resource { return &userApiTokenResource{} }

type userApiTokenResource struct {
	client *client.Client
}

type userApiTokenResourceModel struct {
	ID            types.String `tfsdk:"id"`
	UserID        types.String `tfsdk:"user_id"`
	Name          types.String `tfsdk:"name"`
	Scopes        types.List   `tfsdk:"scopes"`
	ExpiresInDays types.Int64  `tfsdk:"expires_in_days"`
	Token         types.String `tfsdk:"token"`
	TokenPrefix   types.String `tfsdk:"token_prefix"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
	LastUsedAt    types.String `tfsdk:"last_used_at"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (r *userApiTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_api_token"
}

func (r *userApiTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Mints an API token for a specific user (`POST /users/{id}/tokens`), which an admin can do on someone else's behalf. Use `artifactkeeper_api_token` for the credential the provider itself authenticates with, and `artifactkeeper_service_account_token` for machine identities. Tokens are immutable; any change forces a new token, and the secret is returned only on creation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the user the token belongs to. Minting for another user requires an admin credential. Changing this forces a new token.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name for the token.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"scopes": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Token scopes. Required here (unlike `artifactkeeper_api_token`, this endpoint has no default). A non-admin caller cannot mint `*`, `admin`, `delete:*` or `write:users`, and a scoped credential cannot mint beyond its own scopes.",
				Validators:          []validator.List{listvalidator.ValueStringsAre(stringvalidator.OneOf(tokenScopes...)), listvalidator.SizeAtLeast(1)},
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"expires_in_days": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Days until expiry (1-365). Omit for a non-expiring token.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The plaintext token. Only known at creation; not recoverable later (and not populated on import).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"token_prefix": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expires_at":   schema.StringAttribute{Computed: true},
			"last_used_at": schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *userApiTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *userApiTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userApiTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, d := listToStringSlice(ctx, plan.Scopes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateUserApiTokenRequest{Name: plan.Name.ValueString(), Scopes: scopes}
	if !plan.ExpiresInDays.IsNull() {
		createReq.ExpiresInDays = plan.ExpiresInDays.ValueInt64Pointer()
	}

	userID := plan.UserID.ValueString()
	created, err := r.client.CreateUserApiToken(ctx, userID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user API token", err.Error())
		return
	}

	// The create response omits scopes/prefix/expiry, so read the metadata back.
	meta, err := r.client.GetUserApiToken(ctx, userID, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading back created user API token", err.Error())
		return
	}

	state, d := userApiTokenToModel(ctx, meta, userID)
	resp.Diagnostics.Append(d...)
	state.Token = types.StringValue(created.Token)
	state.ExpiresInDays = plan.ExpiresInDays
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *userApiTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userApiTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta, err := r.client.GetUserApiToken(ctx, state.UserID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user API token", err.Error())
		return
	}

	refreshed, d := userApiTokenToModel(ctx, meta, state.UserID.ValueString())
	resp.Diagnostics.Append(d...)
	refreshed.Token = state.Token // secret preserved; never returned by the API
	refreshed.ExpiresInDays = state.ExpiresInDays
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update never runs: every field forces replacement. Here to satisfy the interface.
func (r *userApiTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userApiTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userApiTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userApiTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteUserApiToken(ctx, state.UserID.ValueString(), state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting user API token", err.Error())
	}
}

// ImportState accepts a composite ID of the form "<user_id>/<token_id>"; the
// plaintext token is not recoverable, so it stays null after import.
func (r *userApiTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the form \"<user_id>/<token_id>\".",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func userApiTokenToModel(ctx context.Context, t *client.ApiToken, userID string) (userApiTokenResourceModel, diag.Diagnostics) {
	scopes, d := stringListValue(ctx, t.Scopes)
	return userApiTokenResourceModel{
		ID:          types.StringValue(t.ID),
		UserID:      types.StringValue(userID),
		Name:        types.StringValue(t.Name),
		Scopes:      scopes,
		TokenPrefix: types.StringValue(t.TokenPrefix),
		ExpiresAt:   stringPointerValue(t.ExpiresAt),
		LastUsedAt:  stringPointerValue(t.LastUsedAt),
		CreatedAt:   types.StringValue(t.CreatedAt),
	}, d
}
