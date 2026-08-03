package provider

import (
	"context"
	"encoding/json"
	"fmt"

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
	RuleType          types.String `tfsdk:"rule_type"`
	Config            types.String `tfsdk:"config"`
	Scope             types.String `tfsdk:"scope"`
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
			"rule_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Evaluation engine: `pattern` (glob match on `package_pattern`, the default), `publisher_trust` (allow-list of trusted publishers), or `popularity` (download-count and typosquat signals). The `publisher_trust` and `popularity` engines read their parameters from `config`.",
				Validators:          []validator.String{stringvalidator.OneOf("pattern", "publisher_trust", "popularity")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"config": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Engine parameters as a JSON object string (use `jsonencode(...)`). Ignored by `pattern` rules. `publisher_trust` reads `trusted_publishers`, `match`, `action`; `popularity` reads `min_downloads`, `typosquat_check`, `max_distance`, `action`, `block_unknown`, `block_typosquat`, `homoglyph_check`, `affix_max_downloads`. Defaults to `{}`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"scope": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Rule scope: `repository` (default; requires `staging_repo_id`) or `global` (instance-wide baseline; `staging_repo_id` must be omitted). Immutable, changing it forces a new rule.",
				Validators:          []validator.String{stringvalidator.OneOf("repository", "global")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
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
	if !plan.RuleType.IsNull() && !plan.RuleType.IsUnknown() {
		createReq.RuleType = plan.RuleType.ValueStringPointer()
	}
	if !plan.Scope.IsNull() && !plan.Scope.IsUnknown() {
		createReq.Scope = plan.Scope.ValueStringPointer()
	}
	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		cfg, err := curationConfigJSON(plan.Config.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid curation rule config", err.Error())
			return
		}
		createReq.Config = cfg
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
			// Re-send the created rule's type/config so the full-replace update
			// doesn't reset a typed rule to a pattern rule.
			RuleType: rule.RuleType,
			Config:   rule.Config,
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

	// rule_type and config are always sent: the API full-replaces, defaulting
	// them to "pattern"/{} when omitted, so omitting them would wipe a typed rule.
	cfg, err := curationConfigJSON(plan.Config.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid curation rule config", err.Error())
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
		RuleType:          plan.RuleType.ValueString(),
		Config:            cfg,
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

// curationConfigJSON validates that s is a JSON object and returns it in
// canonical form for the request body. An empty string is treated as `{}`.
func curationConfigJSON(s string) (json.RawMessage, error) {
	if s == "" {
		return json.RawMessage("{}"), nil
	}
	canon, err := canonicalJSON(s)
	if err != nil {
		return nil, fmt.Errorf("config must be a JSON object string: %w", err)
	}
	if canon[0] != '{' {
		return nil, fmt.Errorf("config must be a JSON object, e.g. jsonencode({...})")
	}
	return json.RawMessage(canon), nil
}

func curationRuleToModel(rule *client.CurationRule) curationRuleResourceModel {
	config := "{}"
	if len(rule.Config) > 0 {
		if canon, err := canonicalJSON(string(rule.Config)); err == nil {
			config = canon
		} else {
			config = string(rule.Config)
		}
	}
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
		RuleType:          types.StringValue(rule.RuleType),
		Config:            types.StringValue(config),
		Scope:             types.StringValue(rule.Scope),
		CreatedBy:         stringPointerValue(rule.CreatedBy),
		CreatedAt:         types.StringValue(rule.CreatedAt),
		UpdatedAt:         types.StringValue(rule.UpdatedAt),
	}
}
