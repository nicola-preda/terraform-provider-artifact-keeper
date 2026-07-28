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

// telemetrySettingsID is the fixed id for this singleton (no per-object id
// server-side), giving Terraform a stable address to import and refresh.
const telemetrySettingsID = "telemetry"

var (
	_ resource.Resource                = (*telemetrySettingsResource)(nil)
	_ resource.ResourceWithConfigure   = (*telemetrySettingsResource)(nil)
	_ resource.ResourceWithImportState = (*telemetrySettingsResource)(nil)
)

func NewTelemetrySettingsResource() resource.Resource { return &telemetrySettingsResource{} }

type telemetrySettingsResource struct {
	client *client.Client
}

type telemetrySettingsResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	ReviewBeforeSend types.Bool   `tfsdk:"review_before_send"`
	ScrubLevel       types.String `tfsdk:"scrub_level"`
	IncludeLogs      types.Bool   `tfsdk:"include_logs"`
}

func (r *telemetrySettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_telemetry_settings"
}

func (r *telemetrySettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Global telemetry and crash-reporting settings, a singleton (one instance regardless of how many times declared; delete is a no-op and doesn't reset the backend). Every field is optional; an unset field keeps the currently stored value.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Fixed identifier for the telemetry settings singleton (always `telemetry`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether telemetry and crash reporting are enabled. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"review_before_send": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether crash reports are held for manual review before submission upstream. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"scrub_level": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "PII scrubbing level applied to crash reports: `minimal` (emails and IPs only), `standard` (also usernames, repo and artifact names, file paths and JWTs) or `aggressive` (all potentially identifying information). Defaults to `standard`.",
				Validators:          []validator.String{stringvalidator.OneOf("minimal", "standard", "aggressive")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"include_logs": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether application logs are attached to crash reports. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *telemetrySettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *telemetrySettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan telemetrySettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Singleton, no create endpoint: POST settings then read them back.
	cfg, err := r.applySettings(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring telemetry settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, telemetryToModel(cfg))...)
}

func (r *telemetrySettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state telemetrySettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Singleton always exists (backend defaults when unset): never RemoveResource.
	cfg, err := r.client.GetTelemetrySettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading telemetry settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, telemetryToModel(cfg))...)
}

func (r *telemetrySettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan telemetrySettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.applySettings(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error updating telemetry settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, telemetryToModel(cfg))...)
}

func (r *telemetrySettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: no delete/reset endpoint. Destroy just stops managing them; backend
	// keeps them.
}

func (r *telemetrySettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applySettings merges the plan onto current settings (unset fields keep their
// stored value), POSTs, and returns the GET read-back. POST body requires all
// four fields.
func (r *telemetrySettingsResource) applySettings(ctx context.Context, m telemetrySettingsResourceModel) (*client.TelemetrySettings, error) {
	cur, err := r.client.GetTelemetrySettings(ctx)
	if err != nil {
		return nil, err
	}

	merged := *cur
	if !m.Enabled.IsNull() && !m.Enabled.IsUnknown() {
		merged.Enabled = m.Enabled.ValueBool()
	}
	if !m.ReviewBeforeSend.IsNull() && !m.ReviewBeforeSend.IsUnknown() {
		merged.ReviewBeforeSend = m.ReviewBeforeSend.ValueBool()
	}
	if !m.ScrubLevel.IsNull() && !m.ScrubLevel.IsUnknown() {
		merged.ScrubLevel = m.ScrubLevel.ValueString()
	}
	if !m.IncludeLogs.IsNull() && !m.IncludeLogs.IsUnknown() {
		merged.IncludeLogs = m.IncludeLogs.ValueBool()
	}

	if _, err := r.client.UpdateTelemetrySettings(ctx, merged); err != nil {
		return nil, err
	}
	return r.client.GetTelemetrySettings(ctx)
}

func telemetryToModel(c *client.TelemetrySettings) telemetrySettingsResourceModel {
	return telemetrySettingsResourceModel{
		ID:               types.StringValue(telemetrySettingsID),
		Enabled:          types.BoolValue(c.Enabled),
		ReviewBeforeSend: types.BoolValue(c.ReviewBeforeSend),
		ScrubLevel:       types.StringValue(c.ScrubLevel),
		IncludeLogs:      types.BoolValue(c.IncludeLogs),
	}
}
