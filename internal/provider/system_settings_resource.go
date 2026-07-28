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

// systemSettingsID is the fixed id for the settings singleton (no per-object id
// server-side), keeping plans and imports stable.
const systemSettingsID = "system"

var (
	_ resource.Resource                = (*systemSettingsResource)(nil)
	_ resource.ResourceWithConfigure   = (*systemSettingsResource)(nil)
	_ resource.ResourceWithImportState = (*systemSettingsResource)(nil)
)

func NewSystemSettingsResource() resource.Resource { return &systemSettingsResource{} }

type systemSettingsResource struct {
	client *client.Client
}

type systemSettingsResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	StorageBackend            types.String `tfsdk:"storage_backend"`
	StoragePath               types.String `tfsdk:"storage_path"`
	Environment               types.String `tfsdk:"environment"`
	AllowAnonymousDownload    types.Bool   `tfsdk:"allow_anonymous_download"`
	MaxUploadSizeBytes        types.Int64  `tfsdk:"max_upload_size_bytes"`
	RetentionDays             types.Int64  `tfsdk:"retention_days"`
	AuditRetentionDays        types.Int64  `tfsdk:"audit_retention_days"`
	BackupRetentionCount      types.Int64  `tfsdk:"backup_retention_count"`
	EdgeStaleThresholdMinutes types.Int64  `tfsdk:"edge_stale_threshold_minutes"`
}

func (r *systemSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_settings"
}

func (r *systemSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Instance-wide Artifact Keeper settings, a singleton (one record always exists; no create/delete). Applying writes the tunable values; destroying only stops Terraform managing them (they persist server-side). Import with any id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Constant `system`. The settings are a singleton with no server-assigned id.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"storage_backend": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Read-only storage backend name, derived from the server's deployment config.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"storage_path": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Read-only storage path, derived from the server's deployment config.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Read-only deployment environment name, sourced from the server's `ENVIRONMENT` config value.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"allow_anonymous_download": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether unauthenticated clients may download artifacts.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"max_upload_size_bytes": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Maximum artifact upload size, in bytes.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"retention_days": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Number of days artifacts are retained.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"audit_retention_days": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Number of days audit-log entries are retained.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"backup_retention_count": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Number of backups to keep before pruning the oldest.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"edge_stale_threshold_minutes": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minutes after which an edge node is considered stale.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *systemSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *systemSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.writeSettings(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring system settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, systemSettingsToModel(out))...)
}

func (r *systemSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Singleton always exists: never RemoveResource, any error is a real failure.
	out, err := r.client.GetSystemSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading system settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, systemSettingsToModel(out))...)
}

func (r *systemSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.writeSettings(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error updating system settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, systemSettingsToModel(out))...)
}

func (r *systemSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: DB-backed singleton, no delete endpoint. Destroy just drops it from
	// state; settings persist server-side.
}

func (r *systemSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Any import id accepted; normalized to the constant id by the next Read.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// writeSettings POSTs the plan's tunable values then GETs them back. GETs
// current settings first for the read-only fields (required in the body) and to
// fill any tunable the config leaves unset.
func (r *systemSettingsResource) writeSettings(ctx context.Context, plan systemSettingsResourceModel) (*client.SystemSettings, error) {
	cur, err := r.client.GetSystemSettings(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := r.client.UpdateSystemSettings(ctx, systemSettingsRequestFromModel(plan, cur)); err != nil {
		return nil, err
	}

	return r.client.GetSystemSettings(ctx)
}

// systemSettingsRequestFromModel builds the POST body from the plan, using
// current server values (cur) for read-only fields and unset tunables.
func systemSettingsRequestFromModel(m systemSettingsResourceModel, cur *client.SystemSettings) client.SystemSettings {
	req := *cur // start from current server state (incl. read-only fields)

	if !m.AllowAnonymousDownload.IsNull() && !m.AllowAnonymousDownload.IsUnknown() {
		req.AllowAnonymousDownload = m.AllowAnonymousDownload.ValueBool()
	}
	if !m.MaxUploadSizeBytes.IsNull() && !m.MaxUploadSizeBytes.IsUnknown() {
		req.MaxUploadSizeBytes = m.MaxUploadSizeBytes.ValueInt64()
	}
	if !m.RetentionDays.IsNull() && !m.RetentionDays.IsUnknown() {
		req.RetentionDays = m.RetentionDays.ValueInt64()
	}
	if !m.AuditRetentionDays.IsNull() && !m.AuditRetentionDays.IsUnknown() {
		req.AuditRetentionDays = m.AuditRetentionDays.ValueInt64()
	}
	if !m.BackupRetentionCount.IsNull() && !m.BackupRetentionCount.IsUnknown() {
		req.BackupRetentionCount = m.BackupRetentionCount.ValueInt64()
	}
	if !m.EdgeStaleThresholdMinutes.IsNull() && !m.EdgeStaleThresholdMinutes.IsUnknown() {
		req.EdgeStaleThresholdMinutes = m.EdgeStaleThresholdMinutes.ValueInt64()
	}
	return req
}

func systemSettingsToModel(s *client.SystemSettings) systemSettingsResourceModel {
	return systemSettingsResourceModel{
		ID:                        types.StringValue(systemSettingsID),
		StorageBackend:            types.StringValue(s.StorageBackend),
		StoragePath:               types.StringValue(s.StoragePath),
		Environment:               types.StringValue(s.Environment),
		AllowAnonymousDownload:    types.BoolValue(s.AllowAnonymousDownload),
		MaxUploadSizeBytes:        types.Int64Value(s.MaxUploadSizeBytes),
		RetentionDays:             types.Int64Value(s.RetentionDays),
		AuditRetentionDays:        types.Int64Value(s.AuditRetentionDays),
		BackupRetentionCount:      types.Int64Value(s.BackupRetentionCount),
		EdgeStaleThresholdMinutes: types.Int64Value(s.EdgeStaleThresholdMinutes),
	}
}
