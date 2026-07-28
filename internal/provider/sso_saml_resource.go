package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*ssoSamlResource)(nil)
	_ resource.ResourceWithConfigure   = (*ssoSamlResource)(nil)
	_ resource.ResourceWithImportState = (*ssoSamlResource)(nil)
)

func NewSsoSamlResource() resource.Resource { return &ssoSamlResource{} }

type ssoSamlResource struct {
	client *client.Client
}

type ssoSamlResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	EntityID                types.String `tfsdk:"entity_id"`
	SsoURL                  types.String `tfsdk:"sso_url"`
	SloURL                  types.String `tfsdk:"slo_url"`
	Certificate             types.String `tfsdk:"certificate"`
	NameIDFormat            types.String `tfsdk:"name_id_format"`
	AttributeMapping        types.Map    `tfsdk:"attribute_mapping"`
	SpEntityID              types.String `tfsdk:"sp_entity_id"`
	SignRequests            types.Bool   `tfsdk:"sign_requests"`
	RequireSignedAssertions types.Bool   `tfsdk:"require_signed_assertions"`
	AdminGroup              types.String `tfsdk:"admin_group"`
	IsEnabled               types.Bool   `tfsdk:"is_enabled"`
	HasCertificate          types.Bool   `tfsdk:"has_certificate"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

func (r *ssoSamlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_saml"
}

func (r *ssoSamlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	boolDefaulted := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: desc,
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A SAML single sign-on provider.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":        schema.StringAttribute{Required: true, MarkdownDescription: "Unique provider name."},
			"entity_id":   schema.StringAttribute{Required: true, MarkdownDescription: "SAML IdP entity ID."},
			"sso_url":     schema.StringAttribute{Required: true, MarkdownDescription: "IdP SSO endpoint URL."},
			"slo_url":     schema.StringAttribute{Optional: true, MarkdownDescription: "IdP single-logout URL."},
			"certificate": schema.StringAttribute{Required: true, Sensitive: true, MarkdownDescription: "IdP certificate (PEM). Not returned by the API."},
			"name_id_format": schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Defaults to the emailAddress NameID format.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"attribute_mapping": schema.MapAttribute{ElementType: types.StringType, Optional: true, Computed: true,
				MarkdownDescription: "SAML attribute mapping (`username_claim`, `email_claim`, `display_name_claim`, `groups_claim`).",
				PlanModifiers:       []planmodifier.Map{mapplanmodifier.UseStateForUnknown()}},
			"sp_entity_id": schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Service Provider entity ID. Defaults to `artifact-keeper`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"sign_requests":             boolDefaulted("Sign AuthnRequests. Defaults to `false`."),
			"require_signed_assertions": boolDefaulted("Require signed assertions. Defaults to `true`."),
			"admin_group":               schema.StringAttribute{Optional: true, MarkdownDescription: "Group mapped to admin."},
			"is_enabled":                boolDefaulted("Defaults to `true`."),
			"has_certificate":           schema.BoolAttribute{Computed: true},
			"created_at":                schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"updated_at":                schema.StringAttribute{Computed: true},
		},
	}
}

func (r *ssoSamlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *ssoSamlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ssoSamlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiReq, d := samlRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.CreateSamlConfig(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SAML provider", err.Error())
		return
	}
	state, d := samlToModel(ctx, cfg)
	resp.Diagnostics.Append(d...)
	state.Certificate = plan.Certificate
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ssoSamlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ssoSamlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetSamlConfig(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading SAML provider", err.Error())
		return
	}
	refreshed, d := samlToModel(ctx, cfg)
	resp.Diagnostics.Append(d...)
	refreshed.Certificate = state.Certificate
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *ssoSamlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ssoSamlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiReq, d := samlRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.UpdateSamlConfig(ctx, plan.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating SAML provider", err.Error())
		return
	}
	state, d := samlToModel(ctx, cfg)
	resp.Diagnostics.Append(d...)
	state.Certificate = plan.Certificate
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ssoSamlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ssoSamlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSamlConfig(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting SAML provider", err.Error())
	}
}

func (r *ssoSamlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func samlRequestFromModel(ctx context.Context, m ssoSamlResourceModel) (client.SamlConfigRequest, diag.Diagnostics) {
	mapping, diags := mapToStringMap(ctx, m.AttributeMapping)
	req := client.SamlConfigRequest{
		Name:             m.Name.ValueStringPointer(),
		EntityID:         m.EntityID.ValueStringPointer(),
		SsoURL:           m.SsoURL.ValueStringPointer(),
		Certificate:      m.Certificate.ValueStringPointer(),
		AttributeMapping: mapping,
	}
	req.SloURL = optionalString(m.SloURL)
	req.NameIDFormat = optionalString(m.NameIDFormat)
	req.SpEntityID = optionalString(m.SpEntityID)
	req.AdminGroup = optionalString(m.AdminGroup)
	if !m.SignRequests.IsNull() && !m.SignRequests.IsUnknown() {
		req.SignRequests = m.SignRequests.ValueBoolPointer()
	}
	if !m.RequireSignedAssertions.IsNull() && !m.RequireSignedAssertions.IsUnknown() {
		req.RequireSignedAssertions = m.RequireSignedAssertions.ValueBoolPointer()
	}
	if !m.IsEnabled.IsNull() && !m.IsEnabled.IsUnknown() {
		req.IsEnabled = m.IsEnabled.ValueBoolPointer()
	}
	return req, diags
}

func samlToModel(ctx context.Context, c *client.SamlConfig) (ssoSamlResourceModel, diag.Diagnostics) {
	mapping, diags := stringMapValue(ctx, c.AttributeMapping)
	return ssoSamlResourceModel{
		ID:                      types.StringValue(c.ID),
		Name:                    types.StringValue(c.Name),
		EntityID:                types.StringValue(c.EntityID),
		SsoURL:                  types.StringValue(c.SsoURL),
		SloURL:                  stringPointerValue(c.SloURL),
		NameIDFormat:            types.StringValue(c.NameIDFormat),
		AttributeMapping:        mapping,
		SpEntityID:              types.StringValue(c.SpEntityID),
		SignRequests:            types.BoolValue(c.SignRequests),
		RequireSignedAssertions: types.BoolValue(c.RequireSignedAssertions),
		AdminGroup:              stringPointerValue(c.AdminGroup),
		IsEnabled:               types.BoolValue(c.IsEnabled),
		HasCertificate:          types.BoolValue(c.HasCertificate),
		CreatedAt:               types.StringValue(c.CreatedAt),
		UpdatedAt:               types.StringValue(c.UpdatedAt),
	}, diags
}
