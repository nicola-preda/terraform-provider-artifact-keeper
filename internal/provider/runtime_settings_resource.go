package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
	ID                  types.String `tfsdk:"id"`
	GuestAccessEnabled  types.Bool   `tfsdk:"guest_access_enabled"`
	GuestAccessSource   types.String `tfsdk:"guest_access_source"`
	GuestAccessEditable types.Bool   `tfsdk:"guest_access_editable"`
	GuestAccessInForce  types.Bool   `tfsdk:"guest_access_in_force"`
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
	s, err := r.client.UpdateRuntimeSettings(ctx, client.UpdateRuntimeSettingsRequest{
		GuestAccessEnabled: plan.GuestAccessEnabled.ValueBoolPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error configuring runtime settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, runtimeSettingsToModel(s))...)
}

func (r *runtimeSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Singleton always exists: never RemoveResource.
	s, err := r.client.GetRuntimeSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading runtime settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, runtimeSettingsToModel(s))...)
}

func (r *runtimeSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan runtimeSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := r.client.UpdateRuntimeSettings(ctx, client.UpdateRuntimeSettingsRequest{
		GuestAccessEnabled: plan.GuestAccessEnabled.ValueBoolPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating runtime settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, runtimeSettingsToModel(s))...)
}

// Delete is a no-op: reopening anonymous access on destroy would be a silent
// security downgrade, so destroy just stops managing it.
func (r *runtimeSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *runtimeSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// runtimeSettingsToModel reads back the STORED value as the configured one, so
// an environment pin does not show up as drift.
func runtimeSettingsToModel(s *client.RuntimeSettings) runtimeSettingsResourceModel {
	configured := s.GuestAccessEnabled
	if s.GuestAccessStored != nil {
		configured = *s.GuestAccessStored
	}
	return runtimeSettingsResourceModel{
		ID:                  types.StringValue(runtimeSettingsID),
		GuestAccessEnabled:  types.BoolValue(configured),
		GuestAccessSource:   types.StringValue(s.GuestAccessSource),
		GuestAccessEditable: types.BoolValue(s.GuestAccessEditable),
		GuestAccessInForce:  types.BoolValue(s.GuestAccessEnabled),
	}
}
