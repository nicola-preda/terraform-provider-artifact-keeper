package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*repoTokenResource)(nil)
	_ resource.ResourceWithConfigure   = (*repoTokenResource)(nil)
	_ resource.ResourceWithImportState = (*repoTokenResource)(nil)
)

// NewRepoTokenResource is the factory registered with the provider.
func NewRepoTokenResource() resource.Resource { return &repoTokenResource{} }

type repoTokenResource struct {
	client *client.Client
}

// repoTokenResourceModel maps the resource schema. Attribute names mirror the
// Artifact Keeper API fields exactly. A repo token is addressed by the
// (repository_key, id) pair.
type repoTokenResourceModel struct {
	ID            types.String `tfsdk:"id"`
	RepositoryKey types.String `tfsdk:"repository_key"`
	Name          types.String `tfsdk:"name"`
	Scopes        types.List   `tfsdk:"scopes"`
	ExpiresInDays types.Int64  `tfsdk:"expires_in_days"`
	Description   types.String `tfsdk:"description"`
	Token         types.String `tfsdk:"token"`
	TokenPrefix   types.String `tfsdk:"token_prefix"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
	LastUsedAt    types.String `tfsdk:"last_used_at"`
	CreatedAt     types.String `tfsdk:"created_at"`
	IsExpired     types.Bool   `tfsdk:"is_expired"`
	IsRevoked     types.Bool   `tfsdk:"is_revoked"`
	CreatedBy     types.String `tfsdk:"created_by"`
}

func (r *repoTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repo_token"
}

func (r *repoTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates an access token scoped to a single repository in Artifact Keeper. Tokens are immutable; any change forces a new token. The secret is returned only on creation (and is not recoverable on import).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Token UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the repository this token is scoped to. Changing this forces a new token.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name for the token. Changing this forces a new token.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"scopes": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Permission scopes for the token, e.g. `[\"read:artifacts\", \"write:artifacts\"]`. Admin-class scopes (`*`, `delete:artifacts`, ...) require an admin caller. Changing this forces a new token.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"expires_in_days": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Days until expiry (1-365). Omit for a non-expiring token. Not returned by the API, so its configured value is preserved in state. Changing this forces a new token.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional human-readable description. Changing this forces a new token.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The plaintext token. Only known at creation; not recoverable later (and not populated on import).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"token_prefix": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Non-secret prefix of the token, shown in listings.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expires_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 expiry timestamp, if the token expires.",
			},
			"last_used_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of the token's last use, if ever used.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_expired": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the token has expired.",
			},
			"is_revoked": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the token has been revoked.",
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Username of the token's creator, if known.",
			},
		},
	}
}

func (r *repoTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repoTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repoTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, d := listToStringSlice(ctx, plan.Scopes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateRepoTokenRequest{Name: plan.Name.ValueString(), Scopes: scopes}
	if !plan.ExpiresInDays.IsNull() {
		createReq.ExpiresInDays = plan.ExpiresInDays.ValueInt64Pointer()
	}
	if !plan.Description.IsNull() {
		createReq.Description = plan.Description.ValueStringPointer()
	}

	key := plan.RepositoryKey.ValueString()
	created, err := r.client.CreateRepoToken(ctx, key, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating repository token", err.Error())
		return
	}

	// The create response omits prefix/scopes/timestamps, so read the metadata back.
	meta, err := r.client.GetRepoToken(ctx, key, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading back created repository token", err.Error())
		return
	}

	state, d := repoTokenToModel(ctx, key, meta)
	resp.Diagnostics.Append(d...)
	state.Token = types.StringValue(created.Token)
	state.ExpiresInDays = plan.ExpiresInDays // create-only input; not returned by the API
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *repoTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repoTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := state.RepositoryKey.ValueString()
	meta, err := r.client.GetRepoToken(ctx, key, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository token", err.Error())
		return
	}

	refreshed, d := repoTokenToModel(ctx, key, meta)
	resp.Diagnostics.Append(d...)
	refreshed.Token = state.Token                 // secret preserved; never returned by the API
	refreshed.ExpiresInDays = state.ExpiresInDays // create-only input; not returned by the API
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update never runs: every field forces replacement. Here to satisfy the interface.
func (r *repoTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repoTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *repoTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repoTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRepoToken(ctx, state.RepositoryKey.ValueString(), state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting repository token", err.Error())
	}
}

// ImportState accepts a composite ID "<repository_key>/<token_id>"; both parts
// are needed because the API addresses a repo token by that pair. The plaintext
// token is only available at creation and cannot be recovered on import.
func (r *repoTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	key, id, ok := strings.Cut(req.ID, "/")
	if !ok || key == "" || id == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected import ID in the format \"<repository_key>/<token_id>\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository_key"), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func repoTokenToModel(ctx context.Context, key string, t *client.RepoToken) (repoTokenResourceModel, diag.Diagnostics) {
	scopes, d := stringListValue(ctx, t.Scopes)
	return repoTokenResourceModel{
		ID:            types.StringValue(t.ID),
		RepositoryKey: types.StringValue(key),
		Name:          types.StringValue(t.Name),
		Scopes:        scopes,
		TokenPrefix:   types.StringValue(t.TokenPrefix),
		ExpiresAt:     stringPointerValue(t.ExpiresAt),
		LastUsedAt:    stringPointerValue(t.LastUsedAt),
		CreatedAt:     types.StringValue(t.CreatedAt),
		IsExpired:     types.BoolValue(t.IsExpired),
		IsRevoked:     types.BoolValue(t.IsRevoked),
		Description:   stringPointerValue(t.Description),
		CreatedBy:     stringPointerValue(t.CreatedBy),
	}, d
}
