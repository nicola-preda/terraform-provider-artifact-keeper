package provider

import (
	"context"

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

// tokenScopes is the canonical API token scope vocabulary the backend accepts
// (token_service.rs ALLOWED_SCOPES). The mint endpoints reject anything else
// with a 400, so the token resources validate against it at plan time.
var tokenScopes = []string{
	"read:artifacts", "write:artifacts", "delete:artifacts", "promote:artifacts",
	"read:repositories", "write:repositories", "delete:repositories",
	"read:users", "write:users", "trigger:sync", "admin", "*",
}

var (
	_ resource.Resource                = (*apiTokenResource)(nil)
	_ resource.ResourceWithConfigure   = (*apiTokenResource)(nil)
	_ resource.ResourceWithImportState = (*apiTokenResource)(nil)
)

func NewApiTokenResource() resource.Resource { return &apiTokenResource{} }

type apiTokenResource struct {
	client *client.Client
}

type apiTokenResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Scopes        types.List   `tfsdk:"scopes"`
	ExpiresInDays types.Int64  `tfsdk:"expires_in_days"`
	Token         types.String `tfsdk:"token"`
	TokenPrefix   types.String `tfsdk:"token_prefix"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
	LastUsedAt    types.String `tfsdk:"last_used_at"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (r *apiTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_token"
}

func (r *apiTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates an API token for the authenticated user (e.g. for peer-to-peer replication credentials). Tokens are immutable; any change forces a new token. The secret is returned only on creation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name for the token.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"scopes": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Token scopes (e.g. `[\"admin\"]` for peering, or `read:artifacts`/`write:artifacts`). Defaults to `[\"read:artifacts\"]`.",
				Validators:          []validator.List{listvalidator.ValueStringsAre(stringvalidator.OneOf(tokenScopes...))},
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace(), listplanmodifier.UseStateForUnknown()},
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

func (r *apiTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *apiTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, d := listToStringSlice(ctx, plan.Scopes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateApiTokenRequest{Name: plan.Name.ValueString(), Scopes: scopes}
	if !plan.ExpiresInDays.IsNull() {
		createReq.ExpiresInDays = plan.ExpiresInDays.ValueInt64Pointer()
	}

	created, err := r.client.CreateApiToken(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating API token", err.Error())
		return
	}

	// The create response omits scopes/prefix/expiry, so read the metadata back.
	meta, err := r.client.GetApiToken(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading back created API token", err.Error())
		return
	}

	state, d := apiTokenToModel(ctx, meta)
	resp.Diagnostics.Append(d...)
	state.Token = types.StringValue(created.Token)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *apiTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta, err := r.client.GetApiToken(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading API token", err.Error())
		return
	}

	refreshed, d := apiTokenToModel(ctx, meta)
	resp.Diagnostics.Append(d...)
	refreshed.Token = state.Token // secret preserved; never returned by the API
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update never runs: every field forces replacement. Here to satisfy the interface.
func (r *apiTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *apiTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteApiToken(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting API token", err.Error())
	}
}

func (r *apiTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func apiTokenToModel(ctx context.Context, t *client.ApiToken) (apiTokenResourceModel, diag.Diagnostics) {
	scopes, d := stringListValue(ctx, t.Scopes)
	return apiTokenResourceModel{
		ID:          types.StringValue(t.ID),
		Name:        types.StringValue(t.Name),
		Scopes:      scopes,
		TokenPrefix: types.StringValue(t.TokenPrefix),
		ExpiresAt:   stringPointerValue(t.ExpiresAt),
		LastUsedAt:  stringPointerValue(t.LastUsedAt),
		CreatedAt:   types.StringValue(t.CreatedAt),
	}, d
}
