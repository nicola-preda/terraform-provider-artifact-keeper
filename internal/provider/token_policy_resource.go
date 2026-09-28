package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

// tokenPolicyID is the fixed id for this singleton (no per-object id
// server-side), giving Terraform a stable address to import and refresh.
const tokenPolicyID = "token_policy"

var (
	_ resource.Resource                = (*tokenPolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*tokenPolicyResource)(nil)
	_ resource.ResourceWithImportState = (*tokenPolicyResource)(nil)
)

func NewTokenPolicyResource() resource.Resource { return &tokenPolicyResource{} }

type tokenPolicyResource struct {
	client *client.Client
}

type tokenPolicyResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	RequireExpiration      types.Bool   `tfsdk:"require_expiration"`
	MinDays                types.Int64  `tfsdk:"min_days"`
	MaxDays                types.Int64  `tfsdk:"max_days"`
	DefaultDays            types.Int64  `tfsdk:"default_days"`
	ApplyToServiceAccounts types.Bool   `tfsdk:"apply_to_service_accounts"`
	Source                 types.String `tfsdk:"source"`
	Editable               types.Bool   `tfsdk:"editable"`
}

func (r *tokenPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token_policy"
}

func (r *tokenPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Mint-time expiration policy for API tokens, a singleton (one instance regardless of how many times declared; delete is a no-op and doesn't relax the backend).\n\n" +
			"The policy is evaluated only when a token is minted, so enabling it never invalidates a token a pipeline already holds. Tokens that predate it keep whatever `expires_at` they were stamped with, including none.\n\n" +
			"`min_days` and `max_days` are always sent, because the backend replaces the whole policy object on write; they are inert while `require_expiration` is `false`. An apply fails with `409` when the `API_TOKEN_EXPIRATION_*` environment variables pin the policy (`editable` is then `false`, and `API_TOKEN_EXPIRATION_REQUIRED` is the variable that arms the pin), and with `400` when `default_days` falls outside `[min_days, max_days]`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Fixed identifier for the token policy singleton (always `token_policy`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"require_expiration": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether newly minted API tokens must carry an expiration. `false` (the backend default) leaves the policy inert.",
			},
			"min_days": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Smallest accepted `expires_in_days`, inclusive. Must be at least 1.",
				Validators:          []validator.Int64{int64validator.Between(1, 3650)},
			},
			"max_days": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Largest accepted `expires_in_days`, inclusive. Must be at least `min_days` and at most 3650, the backend's absolute ceiling.",
				Validators:          []validator.Int64{int64validator.Between(1, 3650)},
			},
			"default_days": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Expiration applied when an enforced mint omits `expires_in_days`, so a client that never sends one keeps working. Omit it to reject such mints instead. Must fall within `[min_days, max_days]`.",
				Validators:          []validator.Int64{int64validator.Between(1, 3650)},
			},
			"apply_to_service_accounts": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether service-account token mints are subject to the policy. Defaults to `false`: those are the credentials CI runs on, so expiring them is an outage on a schedule.",
			},
			"source": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Where the policy in force comes from: `database` (the stored setting, which this resource owns) or `environment` (the `API_TOKEN_EXPIRATION_*` variables pin it and writes are refused).",
			},
			"editable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the policy can be changed over the API. `false` when `source` is `environment`.",
			},
		},
	}
}

func (r *tokenPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *tokenPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tokenPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Singleton, no create endpoint: PUT then store what the PUT echoes back.
	cfg, err := r.client.UpdateTokenPolicy(ctx, client.UpdateTokenPolicyRequest{Policy: tokenPolicyFromModel(plan)})
	if err != nil {
		resp.Diagnostics.AddError("Error configuring API token expiration policy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, tokenPolicyToModel(cfg))...)
}

func (r *tokenPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tokenPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Singleton always exists (the backend defaults to an inert policy): never
	// RemoveResource.
	cfg, err := r.client.GetTokenPolicy(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading API token expiration policy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, tokenPolicyToModel(cfg))...)
}

func (r *tokenPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan tokenPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.UpdateTokenPolicy(ctx, client.UpdateTokenPolicyRequest{Policy: tokenPolicyFromModel(plan)})
	if err != nil {
		resp.Diagnostics.AddError("Error updating API token expiration policy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, tokenPolicyToModel(cfg))...)
}

// Delete is a no-op: relaxing the policy on destroy would be a silent security
// downgrade, so destroy just stops managing it.
func (r *tokenPolicyResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *tokenPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func tokenPolicyFromModel(m tokenPolicyResourceModel) client.ApiTokenExpiryPolicy {
	p := client.ApiTokenExpiryPolicy{
		RequireExpiration:      m.RequireExpiration.ValueBool(),
		MinDays:                m.MinDays.ValueInt64(),
		MaxDays:                m.MaxDays.ValueInt64(),
		ApplyToServiceAccounts: m.ApplyToServiceAccounts.ValueBool(),
	}
	if !m.DefaultDays.IsNull() && !m.DefaultDays.IsUnknown() {
		p.DefaultDays = m.DefaultDays.ValueInt64Pointer()
	}
	return p
}

func tokenPolicyToModel(c *client.TokenPolicy) tokenPolicyResourceModel {
	return tokenPolicyResourceModel{
		ID:                     types.StringValue(tokenPolicyID),
		RequireExpiration:      types.BoolValue(c.Policy.RequireExpiration),
		MinDays:                types.Int64Value(c.Policy.MinDays),
		MaxDays:                types.Int64Value(c.Policy.MaxDays),
		DefaultDays:            int64PointerValue(c.Policy.DefaultDays),
		ApplyToServiceAccounts: types.BoolValue(c.Policy.ApplyToServiceAccounts),
		Source:                 types.StringValue(c.Source),
		Editable:               types.BoolValue(c.Editable),
	}
}
