package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

// runtimeSettingsID is the fixed id for this singleton.
const runtimeSettingsID = "runtime_settings"

var (
	_ resource.Resource                = (*runtimeSettingsResource)(nil)
	_ resource.ResourceWithConfigure   = (*runtimeSettingsResource)(nil)
	_ resource.ResourceWithImportState = (*runtimeSettingsResource)(nil)
)

func NewRuntimeSettingsResource() resource.Resource { return &runtimeSettingsResource{} }

type runtimeSettingsResource struct {
	client *client.Client
}

type runtimeSettingsResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	GuestAccessEnabled    types.Bool   `tfsdk:"guest_access_enabled"`
	GuestAccessSource     types.String `tfsdk:"guest_access_source"`
	GuestAccessEditable   types.Bool   `tfsdk:"guest_access_editable"`
	GuestAccessInForce    types.Bool   `tfsdk:"guest_access_in_force"`
	HiddenNavItems        types.Set    `tfsdk:"hidden_nav_items"`
	UpstreamJobsPaused    types.Bool   `tfsdk:"upstream_jobs_paused"`
	UpstreamPausedSource  types.String `tfsdk:"upstream_jobs_paused_source"`
	UpstreamPausedInForce types.Bool   `tfsdk:"upstream_jobs_paused_in_force"`
}

func (r *runtimeSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runtime_settings"
}

func (r *runtimeSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Settings an administrator changes at runtime, without a restart (Artifact Keeper 1.11.0), a singleton; delete is a no-op.\n\n" +
			"`AK_GUEST_ACCESS_ENABLED`, when set explicitly, still wins: the write is stored and takes over once the variable is unset, and `guest_access_editable` is `false` meanwhile. " +
			"To move an instance that runs with `AK_GUEST_ACCESS_ENABLED=false` to management here without reopening anonymous access, apply `guest_access_enabled = false` first, then remove the variable.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Fixed identifier for the singleton (always `runtime_settings`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"guest_access_enabled": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Allow (`true`) or refuse (`false`) anonymous access server-wide. Refusing it also makes creating a public repository a `400`.",
			},
			"guest_access_source": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Where the value in force comes from: `environment`, `database` or `default`.",
			},
			"guest_access_editable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "`false` while `AK_GUEST_ACCESS_ENABLED` pins the value.",
			},
			"guest_access_in_force": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "The value the server enforces right now, which differs from `guest_access_enabled` while the environment pins it.",
			},
			"hidden_nav_items": schema.SetAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Web UI navigation entries hidden for every user, as route paths such as `/peers` (Artifact Keeper 1.12.0, #4574). Display only: a hidden page stays reachable by URL and keeps its permission checks. Each entry is 1 to 128 characters of ASCII letters, digits, `/`, `-`, `_` or `.`, at most 200. An empty set shows every entry; left unset, the server's list is kept.",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"upstream_jobs_paused": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Pause (`true`) or resume (`false`) every scheduled job that contacts an upstream on its own: curation sync, environment re-evaluation, the npm replication feed, peer replication transfers and scheduled mirror passes (Artifact Keeper 1.12.0, #5288). `UPSTREAM_JOBS_PAUSED=true` keeps them paused whatever is stored. Left unset, the stored value is kept.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"upstream_jobs_paused_source": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Where the pause in force comes from: `environment`, `database`, `default` or `unavailable` (the setting could not be read, jobs are treated as paused).",
			},
			"upstream_jobs_paused_in_force": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether upstream jobs are paused right now, which differs from `upstream_jobs_paused` while the environment pins it.",
			},
		},
	}
}

func (r *runtimeSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *runtimeSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan runtimeSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Singleton, no create endpoint: PATCH then store what it echoes back.
	body, diags := runtimeSettingsRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := r.client.UpdateRuntimeSettings(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring runtime settings", err.Error())
		return
	}
	m, diags := runtimeSettingsToModel(ctx, s)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *runtimeSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Singleton always exists: never RemoveResource.
	s, err := r.client.GetRuntimeSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading runtime settings", err.Error())
		return
	}
	m, diags := runtimeSettingsToModel(ctx, s)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *runtimeSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan runtimeSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, diags := runtimeSettingsRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := r.client.UpdateRuntimeSettings(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating runtime settings", err.Error())
		return
	}
	m, diags := runtimeSettingsToModel(ctx, s)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

// Delete is a no-op: reopening anonymous access on destroy would be a silent
// security downgrade, so destroy just stops managing it.
func (r *runtimeSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *runtimeSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// runtimeSettingsRequest sends only what the config sets, so an unset
// attribute leaves the server's value alone
func runtimeSettingsRequest(ctx context.Context, plan runtimeSettingsResourceModel) (client.UpdateRuntimeSettingsRequest, diag.Diagnostics) {
	body := client.UpdateRuntimeSettingsRequest{
		GuestAccessEnabled: plan.GuestAccessEnabled.ValueBoolPointer(),
		UpstreamJobsPaused: optionalBool(plan.UpstreamJobsPaused),
	}
	if plan.HiddenNavItems.IsNull() || plan.HiddenNavItems.IsUnknown() {
		return body, nil
	}
	items, diags := setToStringSlice(ctx, plan.HiddenNavItems)
	if items == nil {
		items = []string{}
	}
	body.HiddenNavItems = &items
	return body, diags
}

// runtimeSettingsToModel reads back the STORED values as the configured ones,
// so an environment pin does not show up as drift.
func runtimeSettingsToModel(ctx context.Context, s *client.RuntimeSettings) (runtimeSettingsResourceModel, diag.Diagnostics) {
	configured := s.GuestAccessEnabled
	if s.GuestAccessStored != nil {
		configured = *s.GuestAccessStored
	}
	paused := s.UpstreamJobsPaused
	if s.UpstreamJobsPausedStored != nil {
		paused = *s.UpstreamJobsPausedStored
	}
	items := s.HiddenNavItems
	if items == nil {
		items = []string{}
	}
	hidden, diags := types.SetValueFrom(ctx, types.StringType, items)
	return runtimeSettingsResourceModel{
		ID:                    types.StringValue(runtimeSettingsID),
		GuestAccessEnabled:    types.BoolValue(configured),
		GuestAccessSource:     types.StringValue(s.GuestAccessSource),
		GuestAccessEditable:   types.BoolValue(s.GuestAccessEditable),
		GuestAccessInForce:    types.BoolValue(s.GuestAccessEnabled),
		HiddenNavItems:        hidden,
		UpstreamJobsPaused:    types.BoolValue(paused),
		UpstreamPausedSource:  types.StringValue(s.UpstreamJobsPausedSource),
		UpstreamPausedInForce: types.BoolValue(s.UpstreamJobsPaused),
	}, diags
}
