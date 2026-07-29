package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*pluginResource)(nil)
	_ resource.ResourceWithConfigure   = (*pluginResource)(nil)
	_ resource.ResourceWithImportState = (*pluginResource)(nil)
)

func NewPluginResource() resource.Resource { return &pluginResource{} }

type pluginResource struct {
	client *client.Client
}

// A WASM plugin installed from a git source. Only the git-install path is
// modelled: zip/local uploads and reload are imperative and out of scope. The
// install source and format_key aren't returned on read, so they're carried in
// state rather than refreshed.
type pluginResourceModel struct {
	ID           types.String `tfsdk:"id"`
	SourceGitURL types.String `tfsdk:"source_git_url"`
	SourceGitRef types.String `tfsdk:"source_git_ref"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Config       types.String `tfsdk:"config"`
	Name         types.String `tfsdk:"name"`
	Version      types.String `tfsdk:"version"`
	DisplayName  types.String `tfsdk:"display_name"`
	Description  types.String `tfsdk:"description"`
	Author       types.String `tfsdk:"author"`
	Homepage     types.String `tfsdk:"homepage"`
	PluginType   types.String `tfsdk:"plugin_type"`
	FormatKey    types.String `tfsdk:"format_key"`
	Status       types.String `tfsdk:"status"`
	ConfigSchema types.String `tfsdk:"config_schema"`
	InstalledAt  types.String `tfsdk:"installed_at"`
	EnabledAt    types.String `tfsdk:"enabled_at"`
}

func (r *pluginResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin"
}

func (r *pluginResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Installs a WASM plugin from a git repository and manages its enabled state and configuration. Only git-sourced plugins are supported; zip/local uploads and reload are imperative actions outside Terraform's model. The backend does not return the install source on read, so changing `source_git_url` or `source_git_ref` forces a reinstall.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Plugin UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_git_url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Git repository URL to install the plugin from. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"source_git_ref": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Git ref (tag, branch, or commit) to install. Defaults to the repository default branch. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the plugin is enabled (active). Set to `false` to install but keep the plugin disabled.",
			},
			"config": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Plugin configuration as a JSON object string (use `jsonencode(...)`). Omit to leave the plugin's config unmanaged.",
			},
			"name":         schema.StringAttribute{Computed: true, MarkdownDescription: "Plugin name."},
			"version":      schema.StringAttribute{Computed: true, MarkdownDescription: "Installed plugin version."},
			"display_name": schema.StringAttribute{Computed: true, MarkdownDescription: "Human-readable plugin name."},
			"description":  schema.StringAttribute{Computed: true, MarkdownDescription: "Plugin description, if any."},
			"author":       schema.StringAttribute{Computed: true, MarkdownDescription: "Plugin author, if declared."},
			"homepage":     schema.StringAttribute{Computed: true, MarkdownDescription: "Plugin homepage, if declared."},
			"plugin_type":  schema.StringAttribute{Computed: true, MarkdownDescription: "Plugin type (e.g. `format`, `validator`)."},
			"format_key": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Format key the plugin registers, if it is a format plugin. Known only at install time.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status":        schema.StringAttribute{Computed: true, MarkdownDescription: "Backend plugin status (`active`, `disabled`, `error`)."},
			"config_schema": schema.StringAttribute{Computed: true, MarkdownDescription: "The plugin's declared config JSON schema, if any."},
			"installed_at":  schema.StringAttribute{Computed: true, MarkdownDescription: "When the plugin was installed (RFC 3339)."},
			"enabled_at":    schema.StringAttribute{Computed: true, MarkdownDescription: "When the plugin was last enabled (RFC 3339), if ever."},
		},
	}
}

func (r *pluginResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *pluginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pluginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ref *string
	if !plan.SourceGitRef.IsNull() && !plan.SourceGitRef.IsUnknown() {
		v := plan.SourceGitRef.ValueString()
		ref = &v
	}

	inst, err := r.client.InstallPluginFromGit(ctx, plan.SourceGitURL.ValueString(), ref)
	if err != nil {
		resp.Diagnostics.AddError("Error installing plugin", err.Error())
		return
	}
	id := inst.PluginID

	// Reconcile enabled state to the desired value (install may leave the
	// plugin either active or disabled depending on the backend).
	cur, err := r.client.GetPlugin(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading plugin after install", err.Error())
		return
	}
	if pluginStatusEnabled(cur.Status) != plan.Enabled.ValueBool() {
		if err := r.client.SetPluginEnabled(ctx, id, plan.Enabled.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Error setting plugin enabled state", err.Error())
			return
		}
	}

	// Apply config only when the user manages it.
	if !plan.Config.IsNull() {
		canon, err := normalizePluginJSON(plan.Config.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid plugin config", fmt.Sprintf("config must be a JSON object string: %s", err.Error()))
			return
		}
		if err := r.client.UpdatePluginConfig(ctx, id, json.RawMessage(canon)); err != nil {
			resp.Diagnostics.AddError("Error setting plugin config", err.Error())
			return
		}
	}

	p, err := r.client.GetPlugin(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading plugin after install", err.Error())
		return
	}

	// Server-owned fields from the read; author-supplied ones (source, enabled,
	// config) stay from the plan so the applied state matches it.
	state := plan
	applyPluginServerFields(&state, p)
	state.FormatKey = types.StringValue(inst.FormatKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *pluginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p, err := r.client.GetPlugin(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading plugin", err.Error())
		return
	}

	applyPluginServerFields(&state, p)
	// Drift detection for the enabled flag comes from the live status.
	state.Enabled = types.BoolValue(pluginStatusEnabled(p.Status))

	// Only reconcile config when it is managed; leave it null otherwise so an
	// unmanaged server default never shows up as a diff.
	if !state.Config.IsNull() {
		cfg, err := r.client.GetPluginConfig(ctx, state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading plugin config", err.Error())
			return
		}
		remote := string(cfg)
		if remote == "" {
			remote = "{}"
		}
		// Preserve the author's formatting when semantically equal; otherwise
		// surface the remote value so drift is visible.
		if !pluginJSONEqual(state.Config.ValueString(), remote) {
			if canon, err := normalizePluginJSON(remote); err == nil {
				state.Config = types.StringValue(canon)
			} else {
				state.Config = types.StringValue(remote)
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *pluginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state pluginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	if !plan.Enabled.Equal(state.Enabled) {
		if err := r.client.SetPluginEnabled(ctx, id, plan.Enabled.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Error setting plugin enabled state", err.Error())
			return
		}
	}

	if !plan.Config.Equal(state.Config) {
		if plan.Config.IsNull() {
			// Unmanaging config: reset to an empty object rather than leaving
			// the last-applied config silently in place.
			if err := r.client.UpdatePluginConfig(ctx, id, json.RawMessage("{}")); err != nil {
				resp.Diagnostics.AddError("Error clearing plugin config", err.Error())
				return
			}
		} else {
			canon, err := normalizePluginJSON(plan.Config.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Invalid plugin config", fmt.Sprintf("config must be a JSON object string: %s", err.Error()))
				return
			}
			if err := r.client.UpdatePluginConfig(ctx, id, json.RawMessage(canon)); err != nil {
				resp.Diagnostics.AddError("Error setting plugin config", err.Error())
				return
			}
		}
	}

	p, err := r.client.GetPlugin(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading plugin after update", err.Error())
		return
	}

	newState := plan
	newState.ID = state.ID
	newState.FormatKey = state.FormatKey
	applyPluginServerFields(&newState, p)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *pluginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UninstallPlugin(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error uninstalling plugin", err.Error())
	}
}

// ImportState brings in an installed plugin by UUID. The install source
// (source_git_url/ref) and format_key are not returned by the API, so they must
// be supplied in configuration; a plan after import will reconcile them.
func (r *pluginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyPluginServerFields sets the server-owned, read-only fields from a GET
// response, leaving author-supplied fields (source, enabled, config) untouched.
func applyPluginServerFields(m *pluginResourceModel, p *client.Plugin) {
	m.ID = types.StringValue(p.ID)
	m.Name = types.StringValue(p.Name)
	m.Version = types.StringValue(p.Version)
	m.DisplayName = types.StringValue(p.DisplayName)
	m.Description = stringPointerValue(p.Description)
	m.Author = stringPointerValue(p.Author)
	m.Homepage = stringPointerValue(p.Homepage)
	m.PluginType = types.StringValue(p.PluginType)
	m.Status = types.StringValue(p.Status)
	if len(p.ConfigSchema) == 0 || string(p.ConfigSchema) == "null" {
		m.ConfigSchema = types.StringNull()
	} else {
		m.ConfigSchema = types.StringValue(string(p.ConfigSchema))
	}
	m.InstalledAt = types.StringValue(p.InstalledAt)
	m.EnabledAt = stringPointerValue(p.EnabledAt)
}

func pluginStatusEnabled(status string) bool { return status == "active" }

// normalizePluginJSON canonicalizes a JSON document (sorted object keys, no
// insignificant whitespace) so it compares stably against Terraform's
// jsonencode output and the backend's stored form.
func normalizePluginJSON(s string) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func pluginJSONEqual(a, b string) bool {
	na, ea := normalizePluginJSON(a)
	nb, eb := normalizePluginJSON(b)
	if ea != nil || eb != nil {
		return a == b
	}
	return na == nb
}
