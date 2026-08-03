package provider

import (
	"context"
	"encoding/json"
	"fmt"
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
	_ resource.Resource                = (*serviceAccountTokenResource)(nil)
	_ resource.ResourceWithConfigure   = (*serviceAccountTokenResource)(nil)
	_ resource.ResourceWithImportState = (*serviceAccountTokenResource)(nil)
)

func NewServiceAccountTokenResource() resource.Resource { return &serviceAccountTokenResource{} }

type serviceAccountTokenResource struct {
	client *client.Client
}

// Addressed by the (service_account_id, id) pair.
type serviceAccountTokenResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
	Name             types.String `tfsdk:"name"`
	Scopes           types.List   `tfsdk:"scopes"`
	ExpiresInDays    types.Int64  `tfsdk:"expires_in_days"`
	Description      types.String `tfsdk:"description"`
	RepositoryIDs    types.List   `tfsdk:"repository_ids"`
	RepoSelector     types.String `tfsdk:"repo_selector"`
	Token            types.String `tfsdk:"token"`
	TokenPrefix      types.String `tfsdk:"token_prefix"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
	LastUsedAt       types.String `tfsdk:"last_used_at"`
	CreatedAt        types.String `tfsdk:"created_at"`
	IsExpired        types.Bool   `tfsdk:"is_expired"`
}

func (r *serviceAccountTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_token"
}

func (r *serviceAccountTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An access token owned by an `artifactkeeper_service_account`. Immutable, any change forces a new token; the secret is returned only on creation (not recoverable on import). Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Token UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_account_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the service account that owns this token. Changing this forces a new token.",
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
				MarkdownDescription: "Permission scopes for the token, e.g. `[\"read:artifacts\", \"write:artifacts\"]`. Changing this forces a new token.",
				Validators:          []validator.List{listvalidator.ValueStringsAre(stringvalidator.OneOf(tokenScopes...))},
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"expires_in_days": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Days until expiry (1-365). Omit for a non-expiring token. Not returned by the API, so its configured value is preserved in state. Changing this forces a new token.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional human-readable description. Not returned by the API, so its configured value is preserved in state. Changing this forces a new token.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repository_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Explicit repository UUIDs the token is restricted to. Mutually exclusive with `repo_selector`. Its configured value is preserved in state. Changing this forces a new token.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"repo_selector": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Dynamic repository selector as a JSON object, e.g. `jsonencode({ match_formats = [\"docker\"] })`. Matched repositories are resolved at auth time, so new repos that match are picked up automatically. Mutually exclusive with `repository_ids`. Use `jsonencode(...)`; its configured value is preserved in state. Changing this forces a new token.",
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
		},
	}
}

func (r *serviceAccountTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *serviceAccountTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceAccountTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, d := listToStringSlice(ctx, plan.Scopes)
	resp.Diagnostics.Append(d...)
	repoIDs, d := listToStringSlice(ctx, plan.RepositoryIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateServiceAccountTokenRequest{
		Name:          plan.Name.ValueString(),
		Scopes:        scopes,
		RepositoryIDs: repoIDs,
	}
	if !plan.ExpiresInDays.IsNull() {
		createReq.ExpiresInDays = plan.ExpiresInDays.ValueInt64Pointer()
	}
	if !plan.Description.IsNull() {
		createReq.Description = plan.Description.ValueStringPointer()
	}
	if !plan.RepoSelector.IsNull() {
		createReq.RepoSelector = json.RawMessage(plan.RepoSelector.ValueString())
	}

	accountID := plan.ServiceAccountID.ValueString()
	created, err := r.client.CreateServiceAccountToken(ctx, accountID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service account token", err.Error())
		return
	}

	// Create response omits prefix/scopes/timestamps; read the metadata back.
	meta, err := r.client.GetServiceAccountToken(ctx, accountID, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading back created service account token", err.Error())
		return
	}

	state, d := serviceAccountTokenToModel(ctx, accountID, meta)
	resp.Diagnostics.Append(d...)
	state.Token = types.StringValue(created.Token)
	// Create-only inputs not returned by the list endpoint; preserve verbatim.
	state.ExpiresInDays = plan.ExpiresInDays
	state.Description = plan.Description
	state.RepositoryIDs = plan.RepositoryIDs
	state.RepoSelector = plan.RepoSelector
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *serviceAccountTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceAccountTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID := state.ServiceAccountID.ValueString()
	meta, err := r.client.GetServiceAccountToken(ctx, accountID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service account token", err.Error())
		return
	}

	refreshed, d := serviceAccountTokenToModel(ctx, accountID, meta)
	resp.Diagnostics.Append(d...)
	// Token never returned by the API; preserve from state, likewise create-only inputs.
	refreshed.Token = state.Token
	refreshed.ExpiresInDays = state.ExpiresInDays
	refreshed.Description = state.Description
	refreshed.RepositoryIDs = state.RepositoryIDs
	refreshed.RepoSelector = state.RepoSelector
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Never runs: every field forces replacement. Satisfies the interface.
func (r *serviceAccountTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceAccountTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *serviceAccountTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceAccountTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteServiceAccountToken(ctx, state.ServiceAccountID.ValueString(), state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting service account token", err.Error())
	}
}

// Composite import ID "<service_account_id>/<token_id>"; both parts address the
// token. Plaintext token can't be recovered on import.
func (r *serviceAccountTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	accountID, id, ok := strings.Cut(req.ID, "/")
	if !ok || accountID == "" || id == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected import ID in the format \"<service_account_id>/<token_id>\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_account_id"), accountID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func serviceAccountTokenToModel(ctx context.Context, accountID string, t *client.ServiceAccountToken) (serviceAccountTokenResourceModel, diag.Diagnostics) {
	scopes, d := stringListValue(ctx, t.Scopes)
	return serviceAccountTokenResourceModel{
		ID:               types.StringValue(t.ID),
		ServiceAccountID: types.StringValue(accountID),
		Name:             types.StringValue(t.Name),
		Scopes:           scopes,
		TokenPrefix:      types.StringValue(t.TokenPrefix),
		ExpiresAt:        stringPointerValue(t.ExpiresAt),
		LastUsedAt:       stringPointerValue(t.LastUsedAt),
		CreatedAt:        types.StringValue(t.CreatedAt),
		IsExpired:        types.BoolValue(t.IsExpired),
	}, d
}
