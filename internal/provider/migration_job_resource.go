package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*migrationJobResource)(nil)
	_ resource.ResourceWithConfigure   = (*migrationJobResource)(nil)
	_ resource.ResourceWithImportState = (*migrationJobResource)(nil)
)

func NewMigrationJobResource() resource.Resource { return &migrationJobResource{} }

type migrationJobResource struct {
	client *client.Client
}

type migrationJobResourceModel struct {
	ID                     types.String  `tfsdk:"id"`
	SourceConnectionID     types.String  `tfsdk:"source_connection_id"`
	JobType                types.String  `tfsdk:"job_type"`
	IncludeRepos           types.List    `tfsdk:"include_repos"`
	ExcludeRepos           types.List    `tfsdk:"exclude_repos"`
	ExcludePaths           types.List    `tfsdk:"exclude_paths"`
	IncludeUsers           types.Bool    `tfsdk:"include_users"`
	IncludeGroups          types.Bool    `tfsdk:"include_groups"`
	IncludePermissions     types.Bool    `tfsdk:"include_permissions"`
	IncludeCachedRemote    types.Bool    `tfsdk:"include_cached_remote"`
	DryRun                 types.Bool    `tfsdk:"dry_run"`
	ConflictResolution     types.String  `tfsdk:"conflict_resolution"`
	ConcurrentTransfers    types.Int64   `tfsdk:"concurrent_transfers"`
	ThrottleDelayMs        types.Int64   `tfsdk:"throttle_delay_ms"`
	VerifyChecksums        types.Bool    `tfsdk:"verify_checksums"`
	DateFrom               types.String  `tfsdk:"date_from"`
	DateTo                 types.String  `tfsdk:"date_to"`
	Status                 types.String  `tfsdk:"status"`
	TotalItems             types.Int64   `tfsdk:"total_items"`
	CompletedItems         types.Int64   `tfsdk:"completed_items"`
	FailedItems            types.Int64   `tfsdk:"failed_items"`
	SkippedItems           types.Int64   `tfsdk:"skipped_items"`
	TotalBytes             types.Int64   `tfsdk:"total_bytes"`
	TransferredBytes       types.Int64   `tfsdk:"transferred_bytes"`
	ProgressPercent        types.Float64 `tfsdk:"progress_percent"`
	EstimatedTimeRemaining types.Int64   `tfsdk:"estimated_time_remaining"`
	CreatedAt              types.String  `tfsdk:"created_at"`
	StartedAt              types.String  `tfsdk:"started_at"`
	FinishedAt             types.String  `tfsdk:"finished_at"`
	ErrorSummary           types.String  `tfsdk:"error_summary"`
}

func (r *migrationJobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_migration_job"
}

func (r *migrationJobResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A migration job against a `migration_source` connection. Terraform creates the job in a **pending** state; **it does not start it**. Start, pause, and cancel are separate imperative operations (run them from the UI or `POST /api/v1/migrations/{id}/start`). The job has no update endpoint, so any change forces a new job. Scope a job to specific repositories with `include_repos`; cached remote (proxy) artifacts are excluded unless `include_cached_remote` is set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_connection_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the `migration_source` connection to migrate from.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"job_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Job type: `full`, `incremental`, or `assessment`. Defaults to `full`.",
				Validators:          []validator.String{stringvalidator.OneOf("full", "incremental", "assessment")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"include_repos": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Repository keys to migrate. Empty migrates everything the connection exposes.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"exclude_repos": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Repository keys to skip.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"exclude_paths": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Artifact path globs to skip.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"include_users": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Migrate users. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"include_groups": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Migrate groups. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"include_permissions": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Migrate permissions. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"include_cached_remote": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Migrate cached remote (proxy) artifacts. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"dry_run": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Plan the migration without transferring anything. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"conflict_resolution": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "How to resolve artifacts that already exist, e.g. `skip` (default) or `overwrite`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"concurrent_transfers": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Parallel transfer workers. Defaults to `4`.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"throttle_delay_ms": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Delay between transfers, in milliseconds. Defaults to `100`.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"verify_checksums": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Verify transferred artifacts against source digests. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"date_from": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Only migrate artifacts modified at or after this RFC 3339 timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"date_to": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Only migrate artifacts modified at or before this RFC 3339 timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Job status (e.g. `pending`, `running`, `completed`, `failed`). Refreshed on read.",
			},
			"total_items":              schema.Int64Attribute{Computed: true, MarkdownDescription: "Total items to migrate. Refreshed on read."},
			"completed_items":          schema.Int64Attribute{Computed: true, MarkdownDescription: "Items migrated so far. Refreshed on read."},
			"failed_items":             schema.Int64Attribute{Computed: true, MarkdownDescription: "Items that failed. Refreshed on read."},
			"skipped_items":            schema.Int64Attribute{Computed: true, MarkdownDescription: "Items skipped. Refreshed on read."},
			"total_bytes":              schema.Int64Attribute{Computed: true, MarkdownDescription: "Total bytes to transfer. Refreshed on read."},
			"transferred_bytes":        schema.Int64Attribute{Computed: true, MarkdownDescription: "Bytes transferred so far. Refreshed on read."},
			"progress_percent":         schema.Float64Attribute{Computed: true, MarkdownDescription: "Migration progress percentage. Refreshed on read."},
			"estimated_time_remaining": schema.Int64Attribute{Computed: true, MarkdownDescription: "Estimated seconds remaining (the backend may leave this null). Refreshed on read."},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"started_at":    schema.StringAttribute{Computed: true, MarkdownDescription: "When the job started, once started. Refreshed on read."},
			"finished_at":   schema.StringAttribute{Computed: true, MarkdownDescription: "When the job finished, once finished. Refreshed on read."},
			"error_summary": schema.StringAttribute{Computed: true, MarkdownDescription: "Error summary for a failed job. Refreshed on read."},
		},
	}
}

func (r *migrationJobResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *migrationJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan migrationJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	includeRepos, d := listToStringSlice(ctx, plan.IncludeRepos)
	resp.Diagnostics.Append(d...)
	excludeRepos, d := listToStringSlice(ctx, plan.ExcludeRepos)
	resp.Diagnostics.Append(d...)
	excludePaths, d := listToStringSlice(ctx, plan.ExcludePaths)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateMigrationRequest{
		SourceConnectionID: plan.SourceConnectionID.ValueString(),
		JobType:            optionalString(plan.JobType),
		Config: client.MigrationConfig{
			IncludeRepos:        includeRepos,
			ExcludeRepos:        excludeRepos,
			ExcludePaths:        excludePaths,
			IncludeUsers:        optionalBool(plan.IncludeUsers),
			IncludeGroups:       optionalBool(plan.IncludeGroups),
			IncludePermissions:  optionalBool(plan.IncludePermissions),
			IncludeCachedRemote: optionalBool(plan.IncludeCachedRemote),
			DryRun:              optionalBool(plan.DryRun),
			ConflictResolution:  optionalString(plan.ConflictResolution),
			ConcurrentTransfers: optionalInt64(plan.ConcurrentTransfers),
			ThrottleDelayMs:     optionalInt64(plan.ThrottleDelayMs),
			VerifyChecksums:     optionalBool(plan.VerifyChecksums),
			DateFrom:            optionalString(plan.DateFrom),
			DateTo:              optionalString(plan.DateTo),
		},
	}

	job, err := r.client.CreateMigrationJob(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating migration job", err.Error())
		return
	}

	applyMigrationJobComputed(&plan, job)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *migrationJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state migrationJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.GetMigrationJob(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading migration job", err.Error())
		return
	}

	applyMigrationJobComputed(&state, job)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update is unreachable (no update endpoint; all inputs force replacement).
func (r *migrationJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan migrationJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *migrationJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state migrationJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteMigrationJob(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting migration job", err.Error())
	}
}

func (r *migrationJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyMigrationJobComputed sets the id + server-managed fields on the model,
// leaving the configured inputs untouched (the API returns config as an opaque
// blob, so inputs are carried from plan/state).
func applyMigrationJobComputed(m *migrationJobResourceModel, job *client.MigrationJob) {
	m.ID = types.StringValue(job.ID)
	m.SourceConnectionID = types.StringValue(job.SourceConnectionID)
	m.JobType = types.StringValue(job.JobType)
	m.Status = types.StringValue(job.Status)
	m.TotalItems = types.Int64Value(job.TotalItems)
	m.CompletedItems = types.Int64Value(job.CompletedItems)
	m.FailedItems = types.Int64Value(job.FailedItems)
	m.SkippedItems = types.Int64Value(job.SkippedItems)
	m.TotalBytes = types.Int64Value(job.TotalBytes)
	m.TransferredBytes = types.Int64Value(job.TransferredBytes)
	m.ProgressPercent = types.Float64Value(job.ProgressPercent)
	m.EstimatedTimeRemaining = int64PointerValue(job.EstimatedTimeRemaining)
	m.CreatedAt = types.StringValue(job.CreatedAt)
	m.StartedAt = stringPointerValue(job.StartedAt)
	m.FinishedAt = stringPointerValue(job.FinishedAt)
	m.ErrorSummary = stringPointerValue(job.ErrorSummary)
}
