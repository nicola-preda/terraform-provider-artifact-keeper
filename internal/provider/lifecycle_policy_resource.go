package provider

import (
	"context"
	"encoding/json"

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
	_ resource.Resource                = (*lifecyclePolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*lifecyclePolicyResource)(nil)
	_ resource.ResourceWithImportState = (*lifecyclePolicyResource)(nil)
)

// NewLifecyclePolicyResource is the factory registered with the provider.
func NewLifecyclePolicyResource() resource.Resource { return &lifecyclePolicyResource{} }

type lifecyclePolicyResource struct {
	client *client.Client
}

type lifecyclePolicyResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	RepositoryID        types.String `tfsdk:"repository_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	PolicyType          types.String `tfsdk:"policy_type"`
	Config              types.String `tfsdk:"config"`
	Priority            types.Int64  `tfsdk:"priority"`
	LastRunAt           types.String `tfsdk:"last_run_at"`
	LastRunItemsRemoved types.Int64  `tfsdk:"last_run_items_removed"`
	CronSchedule        types.String `tfsdk:"cron_schedule"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func (r *lifecyclePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lifecycle_policy"
}

func (r *lifecyclePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An artifact lifecycle (retention) policy. This resource manages the definition only; executing a policy, which deletes matching artifacts, is a separate operation. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Lifecycle policy UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UUID of the repository this policy applies to. Omit for a global policy. The `max_versions` and `size_quota_bytes` policy types require a repository. Changing this forces a new policy.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable policy name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Free-form description of the policy.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the policy runs on the schedule / `execute-all` pass. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"policy_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Retention strategy: `max_age_days`, `max_versions`, `no_downloads_days`, `tag_pattern_keep`, `tag_pattern_delete`, or `size_quota_bytes`. Changing this forces a new policy.",
				Validators: []validator.String{stringvalidator.OneOf(
					"max_age_days",
					"max_versions",
					"no_downloads_days",
					"tag_pattern_keep",
					"tag_pattern_delete",
					"size_quota_bytes",
				)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"config": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Policy configuration as a JSON object, e.g. `jsonencode({ max_age_days = 90 })`. The required keys depend on `policy_type`. Use `jsonencode(...)` so the encoding matches the API's normalized form.",
			},
			"priority": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Evaluation priority; lower numbers run first. Defaults to `0`.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"last_run_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of the last execution, if any.",
			},
			"last_run_items_removed": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of artifacts removed in the last execution.",
			},
			"cron_schedule": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional cron expression controlling automatic execution. When omitted, the policy runs only when executed manually.",
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

func (r *lifecyclePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *lifecyclePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan lifecyclePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateLifecyclePolicyRequest{
		Name:       plan.Name.ValueString(),
		PolicyType: plan.PolicyType.ValueString(),
		Config:     json.RawMessage(plan.Config.ValueString()),
	}
	if !plan.RepositoryID.IsNull() {
		createReq.RepositoryID = plan.RepositoryID.ValueStringPointer()
	}
	if !plan.Description.IsNull() {
		createReq.Description = plan.Description.ValueStringPointer()
	}
	if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
		createReq.Priority = plan.Priority.ValueInt64Pointer()
	}
	if !plan.CronSchedule.IsNull() {
		createReq.CronSchedule = plan.CronSchedule.ValueStringPointer()
	}

	policy, err := r.client.CreateLifecyclePolicy(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating lifecycle policy", err.Error())
		return
	}

	// `enabled` is not accepted on create (new policies are always enabled).
	// Apply the desired value with a follow-up partial update when different.
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() && plan.Enabled.ValueBool() != policy.Enabled {
		policy, err = r.client.UpdateLifecyclePolicy(ctx, policy.ID, client.UpdateLifecyclePolicyRequest{
			Enabled: plan.Enabled.ValueBoolPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error setting lifecycle policy enabled state", err.Error())
			return
		}
	}

	state := lifecyclePolicyToModel(policy)
	state.Config = plan.Config // preserve the configured JSON encoding verbatim
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *lifecyclePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state lifecyclePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetLifecyclePolicy(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading lifecycle policy", err.Error())
		return
	}

	refreshed := lifecyclePolicyToModel(policy)
	// Keep the configured JSON encoding when it is semantically equal to what the
	// API returns, so cosmetic differences (whitespace, key order) don't surface
	// as perpetual diffs. Only overwrite it when the config genuinely drifted.
	canon := func(s string) string {
		var v interface{}
		if json.Unmarshal([]byte(s), &v) != nil {
			return s
		}
		b, err := json.Marshal(v)
		if err != nil {
			return s
		}
		return string(b)
	}
	if !state.Config.IsNull() && canon(state.Config.ValueString()) == canon(refreshed.Config.ValueString()) {
		refreshed.Config = state.Config
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *lifecyclePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan lifecyclePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateLifecyclePolicyRequest{
		Name:   plan.Name.ValueStringPointer(),
		Config: json.RawMessage(plan.Config.ValueString()),
	}
	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueStringPointer()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		updateReq.Enabled = plan.Enabled.ValueBoolPointer()
	}
	if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
		updateReq.Priority = plan.Priority.ValueInt64Pointer()
	}
	if !plan.CronSchedule.IsNull() {
		updateReq.CronSchedule = plan.CronSchedule.ValueStringPointer()
	}

	policy, err := r.client.UpdateLifecyclePolicy(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating lifecycle policy", err.Error())
		return
	}

	state := lifecyclePolicyToModel(policy)
	state.Config = plan.Config // preserve the configured JSON encoding verbatim
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *lifecyclePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state lifecyclePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteLifecyclePolicy(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting lifecycle policy", err.Error())
	}
}

func (r *lifecyclePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func lifecyclePolicyToModel(p *client.LifecyclePolicy) lifecyclePolicyResourceModel {
	return lifecyclePolicyResourceModel{
		ID:                  types.StringValue(p.ID),
		RepositoryID:        stringPointerValue(p.RepositoryID),
		Name:                types.StringValue(p.Name),
		Description:         stringPointerValue(p.Description),
		Enabled:             types.BoolValue(p.Enabled),
		PolicyType:          types.StringValue(p.PolicyType),
		Config:              types.StringValue(string(p.Config)),
		Priority:            types.Int64Value(p.Priority),
		LastRunAt:           stringPointerValue(p.LastRunAt),
		LastRunItemsRemoved: int64PointerValue(p.LastRunItemsRemoved),
		CronSchedule:        stringPointerValue(p.CronSchedule),
		CreatedAt:           types.StringValue(p.CreatedAt),
		UpdatedAt:           types.StringValue(p.UpdatedAt),
	}
}
