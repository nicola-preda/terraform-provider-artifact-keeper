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
	_ resource.Resource                = (*ciOidcProviderResource)(nil)
	_ resource.ResourceWithConfigure   = (*ciOidcProviderResource)(nil)
	_ resource.ResourceWithImportState = (*ciOidcProviderResource)(nil)
)

func NewCiOidcProviderResource() resource.Resource { return &ciOidcProviderResource{} }

type ciOidcProviderResource struct {
	client *client.Client
}

type ciOidcProviderResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ProviderType types.String `tfsdk:"provider_type"`
	IssuerURL    types.String `tfsdk:"issuer_url"`
	Audience     types.String `tfsdk:"audience"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	MappingCount types.Int64  `tfsdk:"mapping_count"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (r *ciOidcProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ci_oidc_provider"
}

func (r *ciOidcProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A CI OIDC provider. Pipelines present a signed OIDC token from the configured `issuer_url`, which Artifact Keeper validates against the issuer's JWKS (no client secret). Identity mappings are managed separately.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Provider UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique display name for the provider.",
			},
			"provider_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Provider flavor (free-form, e.g. `generic`, `github`, `gitlab`); only affects the derived username format. Defaults to `generic`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"issuer_url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "OIDC issuer/discovery URL of the CI platform (used to fetch JWKS and validate the `iss` claim).",
			},
			"audience": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Expected `aud` claim in CI tokens. Defaults to `artifact-keeper`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the provider is enabled. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"mapping_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of identity mappings attached to this provider.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *ciOidcProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *ciOidcProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ciOidcProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.CreateCiOidcProvider(ctx, ciOidcRequestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating CI OIDC provider", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ciOidcToModel(cfg))...)
}

func (r *ciOidcProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ciOidcProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetCiOidcProvider(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading CI OIDC provider", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ciOidcToModel(cfg))...)
}

func (r *ciOidcProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ciOidcProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.UpdateCiOidcProvider(ctx, plan.ID.ValueString(), ciOidcRequestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating CI OIDC provider", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ciOidcToModel(cfg))...)
}

func (r *ciOidcProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ciOidcProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCiOidcProvider(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting CI OIDC provider", err.Error())
	}
}

func (r *ciOidcProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func ciOidcRequestFromModel(m ciOidcProviderResourceModel) client.CiOidcProviderRequest {
	req := client.CiOidcProviderRequest{
		Name:      m.Name.ValueStringPointer(),
		IssuerURL: m.IssuerURL.ValueStringPointer(),
	}
	if !m.ProviderType.IsNull() && !m.ProviderType.IsUnknown() {
		req.ProviderType = m.ProviderType.ValueStringPointer()
	}
	if !m.Audience.IsNull() && !m.Audience.IsUnknown() {
		req.Audience = m.Audience.ValueStringPointer()
	}
	if !m.IsEnabled.IsNull() && !m.IsEnabled.IsUnknown() {
		req.IsEnabled = m.IsEnabled.ValueBoolPointer()
	}
	return req
}

func ciOidcToModel(c *client.CiOidcProvider) ciOidcProviderResourceModel {
	return ciOidcProviderResourceModel{
		ID:           types.StringValue(c.ID),
		Name:         types.StringValue(c.Name),
		ProviderType: types.StringValue(c.ProviderType),
		IssuerURL:    types.StringValue(c.IssuerURL),
		Audience:     types.StringValue(c.Audience),
		IsEnabled:    types.BoolValue(c.IsEnabled),
		MappingCount: types.Int64Value(c.MappingCount),
		CreatedAt:    types.StringValue(c.CreatedAt),
		UpdatedAt:    types.StringValue(c.UpdatedAt),
	}
}
