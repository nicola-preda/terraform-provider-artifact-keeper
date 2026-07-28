package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*ssoLdapResource)(nil)
	_ resource.ResourceWithConfigure   = (*ssoLdapResource)(nil)
	_ resource.ResourceWithImportState = (*ssoLdapResource)(nil)
)

func NewSsoLdapResource() resource.Resource { return &ssoLdapResource{} }

type ssoLdapResource struct {
	client *client.Client
}

type ssoLdapResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	ServerURL            types.String `tfsdk:"server_url"`
	BindDN               types.String `tfsdk:"bind_dn"`
	BindPassword         types.String `tfsdk:"bind_password"`
	UserBaseDN           types.String `tfsdk:"user_base_dn"`
	UserFilter           types.String `tfsdk:"user_filter"`
	GroupBaseDN          types.String `tfsdk:"group_base_dn"`
	GroupFilter          types.String `tfsdk:"group_filter"`
	EmailAttribute       types.String `tfsdk:"email_attribute"`
	DisplayNameAttribute types.String `tfsdk:"display_name_attribute"`
	UsernameAttribute    types.String `tfsdk:"username_attribute"`
	GroupsAttribute      types.String `tfsdk:"groups_attribute"`
	AdminGroupDN         types.String `tfsdk:"admin_group_dn"`
	UseStartTLS          types.Bool   `tfsdk:"use_starttls"`
	IsEnabled            types.Bool   `tfsdk:"is_enabled"`
	Priority             types.Int64  `tfsdk:"priority"`
	HasBindPassword      types.Bool   `tfsdk:"has_bind_password"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func (r *ssoLdapResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_ldap"
}

func (r *ssoLdapResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optComputedStr := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: desc,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "An LDAP single sign-on provider.",
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":                   schema.StringAttribute{Required: true, MarkdownDescription: "Unique provider name."},
			"server_url":             schema.StringAttribute{Required: true, MarkdownDescription: "LDAP server URL, e.g. `ldaps://ldap.example.com:636`."},
			"bind_dn":                schema.StringAttribute{Optional: true, MarkdownDescription: "Service account DN for search-then-bind."},
			"bind_password":          schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Password for bind_dn. Not returned by the API."},
			"user_base_dn":           schema.StringAttribute{Required: true, MarkdownDescription: "Base DN for user search."},
			"user_filter":            optComputedStr("User search filter. Defaults to `(uid={0})`."),
			"group_base_dn":          schema.StringAttribute{Optional: true},
			"group_filter":           schema.StringAttribute{Optional: true},
			"email_attribute":        optComputedStr("Defaults to `mail`."),
			"display_name_attribute": optComputedStr("Defaults to `cn`."),
			"username_attribute":     optComputedStr("Defaults to `uid`."),
			"groups_attribute":       optComputedStr("Defaults to `memberOf`."),
			"admin_group_dn":         schema.StringAttribute{Optional: true, MarkdownDescription: "DN of the group mapped to admin."},
			"use_starttls":           schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: "Defaults to `false`.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
			"is_enabled":             schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: "Defaults to `true`.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
			"priority":               schema.Int64Attribute{Optional: true, Computed: true, MarkdownDescription: "Precedence among LDAP providers. Defaults to `0`.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"has_bind_password":      schema.BoolAttribute{Computed: true},
			"created_at":             schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"updated_at":             schema.StringAttribute{Computed: true},
		},
	}
}

func (r *ssoLdapResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *ssoLdapResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ssoLdapResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.CreateLdapConfig(ctx, ldapRequestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating LDAP provider", err.Error())
		return
	}
	state := ldapToModel(cfg)
	state.BindPassword = plan.BindPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ssoLdapResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ssoLdapResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetLdapConfig(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading LDAP provider", err.Error())
		return
	}
	refreshed := ldapToModel(cfg)
	refreshed.BindPassword = state.BindPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *ssoLdapResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ssoLdapResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.UpdateLdapConfig(ctx, plan.ID.ValueString(), ldapRequestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating LDAP provider", err.Error())
		return
	}
	state := ldapToModel(cfg)
	state.BindPassword = plan.BindPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ssoLdapResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ssoLdapResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteLdapConfig(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting LDAP provider", err.Error())
	}
}

func (r *ssoLdapResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func ldapRequestFromModel(m ssoLdapResourceModel) client.LdapConfigRequest {
	req := client.LdapConfigRequest{
		Name:       m.Name.ValueStringPointer(),
		ServerURL:  m.ServerURL.ValueStringPointer(),
		UserBaseDN: m.UserBaseDN.ValueStringPointer(),
	}
	req.BindDN = optionalString(m.BindDN)
	req.BindPassword = optionalString(m.BindPassword)
	req.UserFilter = optionalString(m.UserFilter)
	req.GroupBaseDN = optionalString(m.GroupBaseDN)
	req.GroupFilter = optionalString(m.GroupFilter)
	req.EmailAttribute = optionalString(m.EmailAttribute)
	req.DisplayNameAttribute = optionalString(m.DisplayNameAttribute)
	req.UsernameAttribute = optionalString(m.UsernameAttribute)
	req.GroupsAttribute = optionalString(m.GroupsAttribute)
	req.AdminGroupDN = optionalString(m.AdminGroupDN)
	if !m.UseStartTLS.IsNull() && !m.UseStartTLS.IsUnknown() {
		req.UseStartTLS = m.UseStartTLS.ValueBoolPointer()
	}
	if !m.IsEnabled.IsNull() && !m.IsEnabled.IsUnknown() {
		req.IsEnabled = m.IsEnabled.ValueBoolPointer()
	}
	if !m.Priority.IsNull() && !m.Priority.IsUnknown() {
		req.Priority = m.Priority.ValueInt64Pointer()
	}
	return req
}

func ldapToModel(c *client.LdapConfig) ssoLdapResourceModel {
	return ssoLdapResourceModel{
		ID:                   types.StringValue(c.ID),
		Name:                 types.StringValue(c.Name),
		ServerURL:            types.StringValue(c.ServerURL),
		BindDN:               stringPointerValue(c.BindDN),
		UserBaseDN:           types.StringValue(c.UserBaseDN),
		UserFilter:           types.StringValue(c.UserFilter),
		GroupBaseDN:          stringPointerValue(c.GroupBaseDN),
		GroupFilter:          stringPointerValue(c.GroupFilter),
		EmailAttribute:       types.StringValue(c.EmailAttribute),
		DisplayNameAttribute: types.StringValue(c.DisplayNameAttribute),
		UsernameAttribute:    types.StringValue(c.UsernameAttribute),
		GroupsAttribute:      types.StringValue(c.GroupsAttribute),
		AdminGroupDN:         stringPointerValue(c.AdminGroupDN),
		UseStartTLS:          types.BoolValue(c.UseStartTLS),
		IsEnabled:            types.BoolValue(c.IsEnabled),
		Priority:             types.Int64Value(c.Priority),
		HasBindPassword:      types.BoolValue(c.HasBindPassword),
		CreatedAt:            types.StringValue(c.CreatedAt),
		UpdatedAt:            types.StringValue(c.UpdatedAt),
	}
}
