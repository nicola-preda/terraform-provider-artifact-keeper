package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*ssoOidcResource)(nil)
	_ resource.ResourceWithConfigure   = (*ssoOidcResource)(nil)
	_ resource.ResourceWithImportState = (*ssoOidcResource)(nil)
)

func NewSsoOidcResource() resource.Resource { return &ssoOidcResource{} }

type ssoOidcResource struct {
	client *client.Client
}

type ssoOidcResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	IssuerURL          types.String `tfsdk:"issuer_url"`
	ClientID           types.String `tfsdk:"client_id"`
	ClientSecret       types.String `tfsdk:"client_secret"`
	Scopes             types.List   `tfsdk:"scopes"`
	AttributeMapping   types.Map    `tfsdk:"attribute_mapping"`
	IsEnabled          types.Bool   `tfsdk:"is_enabled"`
	AutoCreateUsers    types.Bool   `tfsdk:"auto_create_users"`
	PkceEnabled        types.Bool   `tfsdk:"pkce_enabled"`
	MapGroupsToGroups  types.Bool   `tfsdk:"map_groups_to_groups"`
	AllowLegacyRsaKeys types.Bool   `tfsdk:"allow_legacy_rsa_keys"`
	HasSecret          types.Bool   `tfsdk:"has_secret"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *ssoOidcResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_oidc"
}

func (r *ssoOidcResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	boolDefaulted := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: desc,
			PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "An OIDC single sign-on provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique display name for the provider.",
			},
			"issuer_url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "OIDC issuer/discovery URL (e.g. your Authelia base URL).",
			},
			"client_id": schema.StringAttribute{
				Required: true,
			},
			"client_secret": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "OIDC client secret. Not returned by the API.",
			},
			"scopes": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "OIDC scopes. Defaults to `[\"openid\", \"profile\", \"email\"]`.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"attribute_mapping": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Claim mapping overrides (e.g. `username_claim`, `email_claim`, `groups_claim`, `admin_group`).",
				PlanModifiers:       []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
			},
			"is_enabled":            boolDefaulted("Whether the provider is enabled. Defaults to `true`."),
			"auto_create_users":     boolDefaulted("Auto-create users on first login. Defaults to `true`."),
			"pkce_enabled":          boolDefaulted("Enable PKCE (S256). Defaults to `true`."),
			"map_groups_to_groups":  boolDefaulted("Sync OIDC groups to Artifact Keeper groups. Defaults to `false`."),
			"allow_legacy_rsa_keys": boolDefaulted("Accept legacy RSA signing keys from the IdP (RSA-SHA1). Defaults to `false`; enable only for an IdP that cannot issue modern keys."),
			"has_secret":            schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether a client secret is configured."},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *ssoOidcResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *ssoOidcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ssoOidcResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, d := oidcRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.CreateOidcConfig(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating OIDC provider", err.Error())
		return
	}

	state, d := oidcToModel(ctx, cfg)
	resp.Diagnostics.Append(d...)
	state.ClientSecret = plan.ClientSecret // API never returns it; preserved from config
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ssoOidcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ssoOidcResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetOidcConfig(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading OIDC provider", err.Error())
		return
	}

	refreshed, d := oidcToModel(ctx, cfg)
	resp.Diagnostics.Append(d...)
	refreshed.ClientSecret = state.ClientSecret // preserved; never returned
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *ssoOidcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ssoOidcResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, d := oidcRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The plan carries the full desired attribute_mapping, so it must be
	// authoritative: replace rather than merge, or keys removed from config
	// would linger server-side. Only when we're actually sending the map.
	if apiReq.AttributeMapping != nil {
		replace := true
		apiReq.AttributeMappingReplace = &replace
	}

	cfg, err := r.client.UpdateOidcConfig(ctx, plan.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating OIDC provider", err.Error())
		return
	}

	state, d := oidcToModel(ctx, cfg)
	resp.Diagnostics.Append(d...)
	state.ClientSecret = plan.ClientSecret
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ssoOidcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ssoOidcResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOidcConfig(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting OIDC provider", err.Error())
	}
}

func (r *ssoOidcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func oidcRequestFromModel(ctx context.Context, m ssoOidcResourceModel) (client.OidcConfigRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	scopes, d := listToStringSlice(ctx, m.Scopes)
	diags.Append(d...)
	mapping, d := mapToStringMap(ctx, m.AttributeMapping)
	diags.Append(d...)

	req := client.OidcConfigRequest{
		Name:             m.Name.ValueStringPointer(),
		IssuerURL:        m.IssuerURL.ValueStringPointer(),
		ClientID:         m.ClientID.ValueStringPointer(),
		ClientSecret:     m.ClientSecret.ValueStringPointer(),
		Scopes:           scopes,
		AttributeMapping: mapping,
	}
	if !m.IsEnabled.IsNull() && !m.IsEnabled.IsUnknown() {
		req.IsEnabled = m.IsEnabled.ValueBoolPointer()
	}
	if !m.AutoCreateUsers.IsNull() && !m.AutoCreateUsers.IsUnknown() {
		req.AutoCreateUsers = m.AutoCreateUsers.ValueBoolPointer()
	}
	if !m.PkceEnabled.IsNull() && !m.PkceEnabled.IsUnknown() {
		req.PkceEnabled = m.PkceEnabled.ValueBoolPointer()
	}
	if !m.MapGroupsToGroups.IsNull() && !m.MapGroupsToGroups.IsUnknown() {
		req.MapGroupsToGroups = m.MapGroupsToGroups.ValueBoolPointer()
	}
	if !m.AllowLegacyRsaKeys.IsNull() && !m.AllowLegacyRsaKeys.IsUnknown() {
		req.AllowLegacyRsaKeys = m.AllowLegacyRsaKeys.ValueBoolPointer()
	}
	return req, diags
}

func oidcToModel(ctx context.Context, c *client.OidcConfig) (ssoOidcResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	scopes, d := stringListValue(ctx, c.Scopes)
	diags.Append(d...)
	mapping, d := stringMapValue(ctx, c.AttributeMapping)
	diags.Append(d...)

	return ssoOidcResourceModel{
		ID:                 types.StringValue(c.ID),
		Name:               types.StringValue(c.Name),
		IssuerURL:          types.StringValue(c.IssuerURL),
		ClientID:           types.StringValue(c.ClientID),
		Scopes:             scopes,
		AttributeMapping:   mapping,
		IsEnabled:          types.BoolValue(c.IsEnabled),
		AutoCreateUsers:    types.BoolValue(c.AutoCreateUsers),
		PkceEnabled:        types.BoolValue(c.PkceEnabled),
		MapGroupsToGroups:  types.BoolValue(c.MapGroupsToGroups),
		AllowLegacyRsaKeys: types.BoolValue(c.AllowLegacyRsaKeys),
		HasSecret:          types.BoolValue(c.HasSecret),
		CreatedAt:          types.StringValue(c.CreatedAt),
		UpdatedAt:          types.StringValue(c.UpdatedAt),
	}, diags
}
