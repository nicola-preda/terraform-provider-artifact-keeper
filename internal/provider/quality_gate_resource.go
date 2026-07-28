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
	_ resource.Resource                = (*qualityGateResource)(nil)
	_ resource.ResourceWithConfigure   = (*qualityGateResource)(nil)
	_ resource.ResourceWithImportState = (*qualityGateResource)(nil)
)

// NewQualityGateResource is the factory registered with the provider.
func NewQualityGateResource() resource.Resource { return &qualityGateResource{} }

type qualityGateResource struct {
	client *client.Client
}

type qualityGateResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	RepositoryID       types.String `tfsdk:"repository_id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	MinHealthScore     types.Int64  `tfsdk:"min_health_score"`
	MinSecurityScore   types.Int64  `tfsdk:"min_security_score"`
	MinQualityScore    types.Int64  `tfsdk:"min_quality_score"`
	MinMetadataScore   types.Int64  `tfsdk:"min_metadata_score"`
	MaxCriticalIssues  types.Int64  `tfsdk:"max_critical_issues"`
	MaxHighIssues      types.Int64  `tfsdk:"max_high_issues"`
	MaxMediumIssues    types.Int64  `tfsdk:"max_medium_issues"`
	RequiredChecks     types.List   `tfsdk:"required_checks"`
	EnforceOnPromotion types.Bool   `tfsdk:"enforce_on_promotion"`
	EnforceOnDownload  types.Bool   `tfsdk:"enforce_on_download"`
	Action             types.String `tfsdk:"action"`
	IsEnabled          types.Bool   `tfsdk:"is_enabled"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *qualityGateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_quality_gate"
}

func (r *qualityGateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optionalScore := func(desc string) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, MarkdownDescription: desc}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A quality gate: health-score thresholds and issue limits enforced on promotion/download. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Quality gate UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UUID of the repository this gate applies to. Omit for a global gate. Changing this forces a new gate.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable gate name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Free-form description of the gate.",
			},
			"min_health_score":    optionalScore("Minimum overall health score (0-100) an artifact must meet."),
			"min_security_score":  optionalScore("Minimum security score (0-100) an artifact must meet."),
			"min_quality_score":   optionalScore("Minimum quality score (0-100) an artifact must meet."),
			"min_metadata_score":  optionalScore("Minimum metadata score (0-100) an artifact must meet."),
			"max_critical_issues": optionalScore("Maximum number of critical issues allowed."),
			"max_high_issues":     optionalScore("Maximum number of high-severity issues allowed."),
			"max_medium_issues":   optionalScore("Maximum number of medium-severity issues allowed."),
			"required_checks": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Check types (e.g. `metadata_completeness`) that must have completed for an artifact to pass. Defaults to an empty list.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"enforce_on_promotion": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the gate is enforced during promotion. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"enforce_on_download": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the gate is enforced on download. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"action": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Action to take when the gate fails: `allow`, `warn`, or `block`. Defaults to `warn`.",
				Validators:          []validator.String{stringvalidator.OneOf("allow", "warn", "block")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the gate is active. Defaults to `true`.",
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

func (r *qualityGateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *qualityGateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan qualityGateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateQualityGateRequest{Name: plan.Name.ValueString()}
	if !plan.RepositoryID.IsNull() {
		createReq.RepositoryID = plan.RepositoryID.ValueStringPointer()
	}
	if !plan.Description.IsNull() {
		createReq.Description = plan.Description.ValueStringPointer()
	}
	if !plan.MinHealthScore.IsNull() {
		createReq.MinHealthScore = plan.MinHealthScore.ValueInt64Pointer()
	}
	if !plan.MinSecurityScore.IsNull() {
		createReq.MinSecurityScore = plan.MinSecurityScore.ValueInt64Pointer()
	}
	if !plan.MinQualityScore.IsNull() {
		createReq.MinQualityScore = plan.MinQualityScore.ValueInt64Pointer()
	}
	if !plan.MinMetadataScore.IsNull() {
		createReq.MinMetadataScore = plan.MinMetadataScore.ValueInt64Pointer()
	}
	if !plan.MaxCriticalIssues.IsNull() {
		createReq.MaxCriticalIssues = plan.MaxCriticalIssues.ValueInt64Pointer()
	}
	if !plan.MaxHighIssues.IsNull() {
		createReq.MaxHighIssues = plan.MaxHighIssues.ValueInt64Pointer()
	}
	if !plan.MaxMediumIssues.IsNull() {
		createReq.MaxMediumIssues = plan.MaxMediumIssues.ValueInt64Pointer()
	}
	if !plan.RequiredChecks.IsNull() && !plan.RequiredChecks.IsUnknown() {
		checks, d := listToStringSlice(ctx, plan.RequiredChecks)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.RequiredChecks = checks
	}
	if !plan.EnforceOnPromotion.IsNull() && !plan.EnforceOnPromotion.IsUnknown() {
		createReq.EnforceOnPromotion = plan.EnforceOnPromotion.ValueBoolPointer()
	}
	if !plan.EnforceOnDownload.IsNull() && !plan.EnforceOnDownload.IsUnknown() {
		createReq.EnforceOnDownload = plan.EnforceOnDownload.ValueBoolPointer()
	}
	if !plan.Action.IsNull() && !plan.Action.IsUnknown() {
		createReq.Action = plan.Action.ValueStringPointer()
	}

	gate, err := r.client.CreateQualityGate(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating quality gate", err.Error())
		return
	}

	// `is_enabled` is not accepted on create (new gates are always enabled).
	// Apply the desired value with a follow-up partial update when different.
	if !plan.IsEnabled.IsNull() && !plan.IsEnabled.IsUnknown() && plan.IsEnabled.ValueBool() != gate.IsEnabled {
		gate, err = r.client.UpdateQualityGate(ctx, gate.ID, client.UpdateQualityGateRequest{
			IsEnabled: plan.IsEnabled.ValueBoolPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error setting quality gate enabled state", err.Error())
			return
		}
	}

	state, d := qualityGateToModel(ctx, gate)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *qualityGateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state qualityGateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gate, err := r.client.GetQualityGate(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading quality gate", err.Error())
		return
	}

	refreshed, d := qualityGateToModel(ctx, gate)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *qualityGateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan qualityGateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateQualityGateRequest{Name: plan.Name.ValueStringPointer()}
	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueStringPointer()
	}
	if !plan.MinHealthScore.IsNull() {
		updateReq.MinHealthScore = plan.MinHealthScore.ValueInt64Pointer()
	}
	if !plan.MinSecurityScore.IsNull() {
		updateReq.MinSecurityScore = plan.MinSecurityScore.ValueInt64Pointer()
	}
	if !plan.MinQualityScore.IsNull() {
		updateReq.MinQualityScore = plan.MinQualityScore.ValueInt64Pointer()
	}
	if !plan.MinMetadataScore.IsNull() {
		updateReq.MinMetadataScore = plan.MinMetadataScore.ValueInt64Pointer()
	}
	if !plan.MaxCriticalIssues.IsNull() {
		updateReq.MaxCriticalIssues = plan.MaxCriticalIssues.ValueInt64Pointer()
	}
	if !plan.MaxHighIssues.IsNull() {
		updateReq.MaxHighIssues = plan.MaxHighIssues.ValueInt64Pointer()
	}
	if !plan.MaxMediumIssues.IsNull() {
		updateReq.MaxMediumIssues = plan.MaxMediumIssues.ValueInt64Pointer()
	}
	if !plan.RequiredChecks.IsNull() && !plan.RequiredChecks.IsUnknown() {
		checks, d := listToStringSlice(ctx, plan.RequiredChecks)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if checks == nil {
			checks = []string{}
		}
		updateReq.RequiredChecks = &checks
	}
	if !plan.EnforceOnPromotion.IsNull() && !plan.EnforceOnPromotion.IsUnknown() {
		updateReq.EnforceOnPromotion = plan.EnforceOnPromotion.ValueBoolPointer()
	}
	if !plan.EnforceOnDownload.IsNull() && !plan.EnforceOnDownload.IsUnknown() {
		updateReq.EnforceOnDownload = plan.EnforceOnDownload.ValueBoolPointer()
	}
	if !plan.Action.IsNull() && !plan.Action.IsUnknown() {
		updateReq.Action = plan.Action.ValueStringPointer()
	}
	if !plan.IsEnabled.IsNull() && !plan.IsEnabled.IsUnknown() {
		updateReq.IsEnabled = plan.IsEnabled.ValueBoolPointer()
	}

	gate, err := r.client.UpdateQualityGate(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating quality gate", err.Error())
		return
	}

	state, d := qualityGateToModel(ctx, gate)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *qualityGateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state qualityGateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteQualityGate(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting quality gate", err.Error())
	}
}

func (r *qualityGateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func qualityGateToModel(ctx context.Context, g *client.QualityGate) (qualityGateResourceModel, diag.Diagnostics) {
	checks, diags := stringListValue(ctx, g.RequiredChecks)
	return qualityGateResourceModel{
		ID:                 types.StringValue(g.ID),
		RepositoryID:       stringPointerValue(g.RepositoryID),
		Name:               types.StringValue(g.Name),
		Description:        stringPointerValue(g.Description),
		MinHealthScore:     int64PointerValue(g.MinHealthScore),
		MinSecurityScore:   int64PointerValue(g.MinSecurityScore),
		MinQualityScore:    int64PointerValue(g.MinQualityScore),
		MinMetadataScore:   int64PointerValue(g.MinMetadataScore),
		MaxCriticalIssues:  int64PointerValue(g.MaxCriticalIssues),
		MaxHighIssues:      int64PointerValue(g.MaxHighIssues),
		MaxMediumIssues:    int64PointerValue(g.MaxMediumIssues),
		RequiredChecks:     checks,
		EnforceOnPromotion: types.BoolValue(g.EnforceOnPromotion),
		EnforceOnDownload:  types.BoolValue(g.EnforceOnDownload),
		Action:             types.StringValue(g.Action),
		IsEnabled:          types.BoolValue(g.IsEnabled),
		CreatedAt:          types.StringValue(g.CreatedAt),
		UpdatedAt:          types.StringValue(g.UpdatedAt),
	}, diags
}
