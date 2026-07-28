package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = (*promotionRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*promotionRuleResource)(nil)
	_ resource.ResourceWithImportState = (*promotionRuleResource)(nil)
)

func NewPromotionRuleResource() resource.Resource { return &promotionRuleResource{} }

type promotionRuleResource struct {
	client *client.Client
}

// promotionRuleResourceModel maps the resource schema. Attribute names mirror
// the Artifact Keeper API fields exactly.
type promotionRuleResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	SourceRepoID       types.String `tfsdk:"source_repo_id"`
	TargetRepoID       types.String `tfsdk:"target_repo_id"`
	IsEnabled          types.Bool   `tfsdk:"is_enabled"`
	MaxCveSeverity     types.String `tfsdk:"max_cve_severity"`
	AllowedLicenses    types.List   `tfsdk:"allowed_licenses"`
	RequireSignature   types.Bool   `tfsdk:"require_signature"`
	MinStagingHours    types.Int64  `tfsdk:"min_staging_hours"`
	MaxArtifactAgeDays types.Int64  `tfsdk:"max_artifact_age_days"`
	MinHealthScore     types.Int64  `tfsdk:"min_health_score"`
	AutoPromote        types.Bool   `tfsdk:"auto_promote"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *promotionRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_promotion_rule"
}

func (r *promotionRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	boolDefaulted := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: desc,
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an auto-promotion rule that gates promotion of artifacts from a staging `source_repo_id` to a release `target_repo_id`. Only admins may manage promotion rules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Promotion rule UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable rule name.",
			},
			"source_repo_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the staging repository artifacts are promoted from. Immutable; changing it forces a new rule.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"target_repo_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the release repository artifacts are promoted to. Immutable; changing it forces a new rule.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"is_enabled": boolDefaulted("Whether the rule is active. Defaults to `true`."),
			"max_cve_severity": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Highest CVE severity tolerated before promotion is blocked (`low`, `medium`, `high`, or `critical`). Setting any value requires the artifact to have a completed scan. Omit to impose no CVE gate.",
			},
			"allowed_licenses": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Allowed SPDX license identifiers. Omit to impose no license gate.",
			},
			"require_signature": boolDefaulted("Require artifacts to be signed. Defaults to `false`."),
			"min_staging_hours": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum hours an artifact must dwell in staging before promotion.",
			},
			"max_artifact_age_days": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum age in days an artifact may have to be eligible for promotion.",
			},
			"min_health_score": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum health score (0-100) an artifact must meet.",
			},
			"auto_promote": boolDefaulted("Automatically promote artifacts that pass every gate. Defaults to `false`."),
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

func (r *promotionRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *promotionRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan promotionRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	licenses, d := listToStringSlice(ctx, plan.AllowedLicenses)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreatePromotionRuleRequest{
		Name:            plan.Name.ValueString(),
		SourceRepoID:    plan.SourceRepoID.ValueString(),
		TargetRepoID:    plan.TargetRepoID.ValueString(),
		MaxCveSeverity:  optionalString(plan.MaxCveSeverity),
		AllowedLicenses: licenses,
	}
	if !plan.IsEnabled.IsNull() && !plan.IsEnabled.IsUnknown() {
		createReq.IsEnabled = plan.IsEnabled.ValueBoolPointer()
	}
	if !plan.RequireSignature.IsNull() && !plan.RequireSignature.IsUnknown() {
		createReq.RequireSignature = plan.RequireSignature.ValueBoolPointer()
	}
	if !plan.AutoPromote.IsNull() && !plan.AutoPromote.IsUnknown() {
		createReq.AutoPromote = plan.AutoPromote.ValueBoolPointer()
	}
	if !plan.MinStagingHours.IsNull() {
		createReq.MinStagingHours = plan.MinStagingHours.ValueInt64Pointer()
	}
	if !plan.MaxArtifactAgeDays.IsNull() {
		createReq.MaxArtifactAgeDays = plan.MaxArtifactAgeDays.ValueInt64Pointer()
	}
	if !plan.MinHealthScore.IsNull() {
		createReq.MinHealthScore = plan.MinHealthScore.ValueInt64Pointer()
	}

	rule, err := r.client.CreatePromotionRule(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating promotion rule", err.Error())
		return
	}

	state, d := promotionRuleToModel(ctx, rule)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *promotionRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state promotionRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetPromotionRule(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading promotion rule", err.Error())
		return
	}

	refreshed, d := promotionRuleToModel(ctx, rule)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *promotionRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan promotionRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	licenses, d := listToStringSlice(ctx, plan.AllowedLicenses)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdatePromotionRuleRequest{
		Name:            plan.Name.ValueStringPointer(),
		MaxCveSeverity:  optionalString(plan.MaxCveSeverity),
		AllowedLicenses: licenses,
	}
	if !plan.IsEnabled.IsNull() && !plan.IsEnabled.IsUnknown() {
		updateReq.IsEnabled = plan.IsEnabled.ValueBoolPointer()
	}
	if !plan.RequireSignature.IsNull() && !plan.RequireSignature.IsUnknown() {
		updateReq.RequireSignature = plan.RequireSignature.ValueBoolPointer()
	}
	if !plan.AutoPromote.IsNull() && !plan.AutoPromote.IsUnknown() {
		updateReq.AutoPromote = plan.AutoPromote.ValueBoolPointer()
	}
	if !plan.MinStagingHours.IsNull() {
		updateReq.MinStagingHours = plan.MinStagingHours.ValueInt64Pointer()
	}
	if !plan.MaxArtifactAgeDays.IsNull() {
		updateReq.MaxArtifactAgeDays = plan.MaxArtifactAgeDays.ValueInt64Pointer()
	}
	if !plan.MinHealthScore.IsNull() {
		updateReq.MinHealthScore = plan.MinHealthScore.ValueInt64Pointer()
	}

	rule, err := r.client.UpdatePromotionRule(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating promotion rule", err.Error())
		return
	}

	state, d := promotionRuleToModel(ctx, rule)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *promotionRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state promotionRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePromotionRule(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting promotion rule", err.Error())
	}
}

func (r *promotionRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func promotionRuleToModel(ctx context.Context, rule *client.PromotionRule) (promotionRuleResourceModel, diag.Diagnostics) {
	licenses, diags := stringListValue(ctx, rule.AllowedLicenses)
	return promotionRuleResourceModel{
		ID:                 types.StringValue(rule.ID),
		Name:               types.StringValue(rule.Name),
		SourceRepoID:       types.StringValue(rule.SourceRepoID),
		TargetRepoID:       types.StringValue(rule.TargetRepoID),
		IsEnabled:          types.BoolValue(rule.IsEnabled),
		MaxCveSeverity:     stringPointerValue(rule.MaxCveSeverity),
		AllowedLicenses:    licenses,
		RequireSignature:   types.BoolValue(rule.RequireSignature),
		MinStagingHours:    int64PointerValue(rule.MinStagingHours),
		MaxArtifactAgeDays: int64PointerValue(rule.MaxArtifactAgeDays),
		MinHealthScore:     int64PointerValue(rule.MinHealthScore),
		AutoPromote:        types.BoolValue(rule.AutoPromote),
		CreatedAt:          types.StringValue(rule.CreatedAt),
		UpdatedAt:          types.StringValue(rule.UpdatedAt),
	}, diags
}
