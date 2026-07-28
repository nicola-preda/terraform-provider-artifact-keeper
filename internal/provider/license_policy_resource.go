package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*licensePolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*licensePolicyResource)(nil)
	_ resource.ResourceWithImportState = (*licensePolicyResource)(nil)
)

func NewLicensePolicyResource() resource.Resource { return &licensePolicyResource{} }

type licensePolicyResource struct {
	client *client.Client
}

type licensePolicyResourceModel struct {
	ID              types.String `tfsdk:"id"`
	RepositoryID    types.String `tfsdk:"repository_id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	AllowedLicenses types.List   `tfsdk:"allowed_licenses"`
	DeniedLicenses  types.List   `tfsdk:"denied_licenses"`
	AllowUnknown    types.Bool   `tfsdk:"allow_unknown"`
	Action          types.String `tfsdk:"action"`
	IsEnabled       types.Bool   `tfsdk:"is_enabled"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func (r *licensePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_policy"
}

func (r *licensePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A license-compliance policy, global (no `repository_id`) or scoped to one repository. No update endpoint (POST upserts), so changing any settable field forces a new policy. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Policy UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Policy name. Unique per repository scope. Changing it forces a new policy.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repository_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UUID of the repository this policy applies to. Omit for a global policy. Changing it forces a new policy.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Human-readable description. Changing it forces a new policy.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"allowed_licenses": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "SPDX license identifiers that are explicitly permitted (e.g. `MIT`, `Apache-2.0`). Changing it forces a new policy.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"denied_licenses": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "SPDX license identifiers that are explicitly denied (e.g. `GPL-3.0-only`). Changing it forces a new policy.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"allow_unknown": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether components with an unrecognized license are allowed. Defaults to `true`. Changing it forces a new policy.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace(), boolplanmodifier.UseStateForUnknown()},
			},
			"action": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Action taken on a policy violation: `allow`, `warn`, or `block`. Defaults to `warn`. Changing it forces a new policy.",
				Validators:          []validator.String{stringvalidator.OneOf("allow", "warn", "block")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the policy is active. Defaults to `true`. Changing it forces a new policy.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace(), boolplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 last-update timestamp.",
			},
		},
	}
}

func (r *licensePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *licensePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan licensePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, d := licensePolicyRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	p, err := r.client.CreateLicensePolicy(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating license policy", err.Error())
		return
	}

	state, d := licensePolicyToModel(ctx, p)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *licensePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state licensePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p, err := r.client.GetLicensePolicy(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading license policy", err.Error())
		return
	}

	refreshed, d := licensePolicyToModel(ctx, p)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Unreachable: no update endpoint, every field forces replacement. Just
// satisfies the interface.
func (r *licensePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan licensePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *licensePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state licensePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteLicensePolicy(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting license policy", err.Error())
	}
}

func (r *licensePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func licensePolicyRequestFromModel(ctx context.Context, m licensePolicyResourceModel) (client.UpsertLicensePolicyRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	allowed, d := listToStringSlice(ctx, m.AllowedLicenses)
	diags.Append(d...)
	denied, d := listToStringSlice(ctx, m.DeniedLicenses)
	diags.Append(d...)

	// Server rejects null (bare Vec, no default); send [] not null.
	if allowed == nil {
		allowed = []string{}
	}
	if denied == nil {
		denied = []string{}
	}

	req := client.UpsertLicensePolicyRequest{
		Name:            m.Name.ValueString(),
		RepositoryID:    optionalString(m.RepositoryID),
		Description:     optionalString(m.Description),
		AllowedLicenses: allowed,
		DeniedLicenses:  denied,
	}
	if !m.AllowUnknown.IsNull() && !m.AllowUnknown.IsUnknown() {
		req.AllowUnknown = m.AllowUnknown.ValueBoolPointer()
	}
	if !m.Action.IsNull() && !m.Action.IsUnknown() {
		req.Action = m.Action.ValueStringPointer()
	}
	if !m.IsEnabled.IsNull() && !m.IsEnabled.IsUnknown() {
		req.IsEnabled = m.IsEnabled.ValueBoolPointer()
	}
	return req, diags
}

func licensePolicyToModel(ctx context.Context, p *client.LicensePolicy) (licensePolicyResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	allowed, d := stringListValue(ctx, p.AllowedLicenses)
	diags.Append(d...)
	denied, d := stringListValue(ctx, p.DeniedLicenses)
	diags.Append(d...)

	return licensePolicyResourceModel{
		ID:              types.StringValue(p.ID),
		RepositoryID:    stringPointerValue(p.RepositoryID),
		Name:            types.StringValue(p.Name),
		Description:     stringPointerValue(p.Description),
		AllowedLicenses: allowed,
		DeniedLicenses:  denied,
		AllowUnknown:    types.BoolValue(p.AllowUnknown),
		Action:          types.StringValue(p.Action),
		IsEnabled:       types.BoolValue(p.IsEnabled),
		CreatedAt:       types.StringValue(p.CreatedAt),
		UpdatedAt:       stringPointerValue(p.UpdatedAt),
	}, diags
}
