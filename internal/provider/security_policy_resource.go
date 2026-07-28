package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*securityPolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*securityPolicyResource)(nil)
	_ resource.ResourceWithImportState = (*securityPolicyResource)(nil)
)

func NewSecurityPolicyResource() resource.Resource { return &securityPolicyResource{} }

type securityPolicyResource struct {
	client *client.Client
}

type securityPolicyResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	RepositoryID       types.String `tfsdk:"repository_id"`
	MaxSeverity        types.String `tfsdk:"max_severity"`
	BlockUnscanned     types.Bool   `tfsdk:"block_unscanned"`
	BlockOnFail        types.Bool   `tfsdk:"block_on_fail"`
	IsEnabled          types.Bool   `tfsdk:"is_enabled"`
	MinStagingHours    types.Int64  `tfsdk:"min_staging_hours"`
	MaxArtifactAgeDays types.Int64  `tfsdk:"max_artifact_age_days"`
	RequireSignature   types.Bool   `tfsdk:"require_signature"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *securityPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_policy"
}

func (r *securityPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A security scan/promotion gating policy, global (no `repository_id`) or scoped to one repository. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Policy UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable policy name.",
			},
			"repository_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UUID of the repository this policy applies to. Omit for a global policy. The API cannot re-target a policy, so changing this forces a new policy.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"max_severity": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Highest finding severity permitted before the policy blocks: one of `critical`, `high`, `medium`, `low` (stored lower-cased by the API).",
				Validators:          []validator.String{stringvalidator.OneOf("critical", "high", "medium", "low")},
			},
			"block_unscanned": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Block artifacts that have not been scanned. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"block_on_fail": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Block artifacts whose scan failed to run.",
			},
			"is_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the policy is active. Not settable at creation (the API enables new policies by default); a value set here is applied via a follow-up update.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"min_staging_hours": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum hours an artifact must stay in staging before promotion. The API cannot clear this back to unset once written; remove the policy to reset it.",
			},
			"max_artifact_age_days": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum age in days for a promotable artifact. The API cannot clear this back to unset once written; remove the policy to reset it.",
			},
			"require_signature": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Require a valid signature on the artifact. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
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

func (r *securityPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *securityPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan securityPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateSecurityPolicyRequest{
		Name:         plan.Name.ValueString(),
		RepositoryID: optionalString(plan.RepositoryID),
		MaxSeverity:  plan.MaxSeverity.ValueString(),
		BlockOnFail:  plan.BlockOnFail.ValueBool(),
	}
	if !plan.BlockUnscanned.IsNull() && !plan.BlockUnscanned.IsUnknown() {
		createReq.BlockUnscanned = plan.BlockUnscanned.ValueBoolPointer()
	}
	if !plan.RequireSignature.IsNull() && !plan.RequireSignature.IsUnknown() {
		createReq.RequireSignature = plan.RequireSignature.ValueBoolPointer()
	}
	if !plan.MinStagingHours.IsNull() {
		createReq.MinStagingHours = plan.MinStagingHours.ValueInt64Pointer()
	}
	if !plan.MaxArtifactAgeDays.IsNull() {
		createReq.MaxArtifactAgeDays = plan.MaxArtifactAgeDays.ValueInt64Pointer()
	}

	p, err := r.client.CreateSecurityPolicy(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating security policy", err.Error())
		return
	}

	// is_enabled is not accepted on create; reconcile via update when the user
	// pinned a value that differs from the server default.
	if !plan.IsEnabled.IsNull() && !plan.IsEnabled.IsUnknown() && plan.IsEnabled.ValueBool() != p.IsEnabled {
		p, err = r.client.UpdateSecurityPolicy(ctx, p.ID, client.UpdateSecurityPolicyRequest{
			IsEnabled: plan.IsEnabled.ValueBoolPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error setting is_enabled on new security policy", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, securityPolicyToModel(p))...)
}

func (r *securityPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state securityPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p, err := r.client.GetSecurityPolicy(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading security policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, securityPolicyToModel(p))...)
}

func (r *securityPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan securityPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateSecurityPolicyRequest{
		Name:             plan.Name.ValueStringPointer(),
		MaxSeverity:      plan.MaxSeverity.ValueStringPointer(),
		BlockUnscanned:   plan.BlockUnscanned.ValueBoolPointer(),
		BlockOnFail:      plan.BlockOnFail.ValueBoolPointer(),
		IsEnabled:        plan.IsEnabled.ValueBoolPointer(),
		RequireSignature: plan.RequireSignature.ValueBoolPointer(),
	}
	if !plan.MinStagingHours.IsNull() {
		updateReq.MinStagingHours = plan.MinStagingHours.ValueInt64Pointer()
	}
	if !plan.MaxArtifactAgeDays.IsNull() {
		updateReq.MaxArtifactAgeDays = plan.MaxArtifactAgeDays.ValueInt64Pointer()
	}

	p, err := r.client.UpdateSecurityPolicy(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating security policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, securityPolicyToModel(p))...)
}

func (r *securityPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state securityPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSecurityPolicy(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting security policy", err.Error())
	}
}

func (r *securityPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func securityPolicyToModel(p *client.SecurityPolicy) securityPolicyResourceModel {
	return securityPolicyResourceModel{
		ID:                 types.StringValue(p.ID),
		Name:               types.StringValue(p.Name),
		RepositoryID:       stringPointerValue(p.RepositoryID),
		MaxSeverity:        types.StringValue(p.MaxSeverity),
		BlockUnscanned:     types.BoolValue(p.BlockUnscanned),
		BlockOnFail:        types.BoolValue(p.BlockOnFail),
		IsEnabled:          types.BoolValue(p.IsEnabled),
		MinStagingHours:    int64PointerValue(p.MinStagingHours),
		MaxArtifactAgeDays: int64PointerValue(p.MaxArtifactAgeDays),
		RequireSignature:   types.BoolValue(p.RequireSignature),
		CreatedAt:          types.StringValue(p.CreatedAt),
		UpdatedAt:          types.StringValue(p.UpdatedAt),
	}
}
