package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*curationRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*curationRuleResource)(nil)
	_ resource.ResourceWithImportState = (*curationRuleResource)(nil)
)

// NewCurationRuleResource is the factory registered with the provider.
func NewCurationRuleResource() resource.Resource { return &curationRuleResource{} }

type curationRuleResource struct {
	client *client.Client
}

type curationRuleResourceModel struct {
	ID                types.String `tfsdk:"id"`
	StagingRepoID     types.String `tfsdk:"staging_repo_id"`
	PackagePattern    types.String `tfsdk:"package_pattern"`
	VersionConstraint types.String `tfsdk:"version_constraint"`
	Architecture      types.String `tfsdk:"architecture"`
	Action            types.String `tfsdk:"action"`
	Priority          types.Int64  `tfsdk:"priority"`
	Reason            types.String `tfsdk:"reason"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	CreatedBy         types.String `tfsdk:"created_by"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (r *curationRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_curation_rule"
}

func (r *curationRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A package curation rule: gates which packages a staging repository accepts from its upstream. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Curation rule UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"staging_repo_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UUID of the staging repository this rule applies to. Omit for a global rule that applies to every repository. Changing this forces a new rule.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"package_pattern": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Glob pattern matched against package names (e.g. `evil-*`).",
			},
			"version_constraint": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Version constraint the rule applies to. Defaults to `*` (any version).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"architecture": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Architecture the rule applies to. Defaults to `*` (any architecture).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"action": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Action to take for matching packages: `allow` or `block`.",
				Validators:          []validator.String{stringvalidator.OneOf("allow", "block")},
			},
			"priority": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Rule priority; lower numbers are evaluated first. Defaults to `100`.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"reason": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable justification recorded with the rule.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the rule is active. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the user who created the rule.",
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

func (r *curationRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *curationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan curationRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateCurationRuleRequest{
		PackagePattern: plan.PackagePattern.ValueString(),
		Action:         plan.Action.ValueString(),
		Reason:         plan.Reason.ValueString(),
	}
	if !plan.StagingRepoID.IsNull() {
		createReq.StagingRepoID = plan.StagingRepoID.ValueStringPointer()
	}
	if !plan.VersionConstraint.IsNull() && !plan.VersionConstraint.IsUnknown() {
		createReq.VersionConstraint = plan.VersionConstraint.ValueStringPointer()
	}
	if !plan.Architecture.IsNull() && !plan.Architecture.IsUnknown() {
		createReq.Architecture = plan.Architecture.ValueStringPointer()
	}
	if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
		createReq.Priority = plan.Priority.ValueInt64Pointer()
	}

	rule, err := r.client.CreateCurationRule(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating curation rule", err.Error())
		return
	}

	// `enabled` is not accepted on create (new rules are always enabled). Apply
	// the desired value with a follow-up full update when the user asked for a
	// different one.
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() && plan.Enabled.ValueBool() != rule.Enabled {
		rule, err = r.client.UpdateCurationRule(ctx, rule.ID, client.UpdateCurationRuleRequest{
			PackagePattern:    rule.PackagePattern,
			VersionConstraint: rule.VersionConstraint,
			Architecture:      rule.Architecture,
			Action:            rule.Action,
			Priority:          rule.Priority,
			Reason:            rule.Reason,
			Enabled:           plan.Enabled.ValueBool(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error setting curation rule enabled state", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, curationRuleToModel(rule))...)
}

func (r *curationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state curationRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetCurationRule(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading curation rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, curationRuleToModel(rule))...)
}

func (r *curationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan curationRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.UpdateCurationRule(ctx, plan.ID.ValueString(), client.UpdateCurationRuleRequest{
		PackagePattern:    plan.PackagePattern.ValueString(),
		VersionConstraint: plan.VersionConstraint.ValueString(),
		Architecture:      plan.Architecture.ValueString(),
		Action:            plan.Action.ValueString(),
		Priority:          plan.Priority.ValueInt64(),
		Reason:            plan.Reason.ValueString(),
		Enabled:           plan.Enabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating curation rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, curationRuleToModel(rule))...)
}

func (r *curationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state curationRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCurationRule(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting curation rule", err.Error())
	}
}

func (r *curationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func curationRuleToModel(rule *client.CurationRule) curationRuleResourceModel {
	return curationRuleResourceModel{
		ID:                types.StringValue(rule.ID),
		StagingRepoID:     stringPointerValue(rule.StagingRepoID),
		PackagePattern:    types.StringValue(rule.PackagePattern),
		VersionConstraint: types.StringValue(rule.VersionConstraint),
		Architecture:      types.StringValue(rule.Architecture),
		Action:            types.StringValue(rule.Action),
		Priority:          types.Int64Value(rule.Priority),
		Reason:            types.StringValue(rule.Reason),
		Enabled:           types.BoolValue(rule.Enabled),
		CreatedBy:         stringPointerValue(rule.CreatedBy),
		CreatedAt:         types.StringValue(rule.CreatedAt),
		UpdatedAt:         types.StringValue(rule.UpdatedAt),
	}
}
