package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*repositoryAllowlistResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryAllowlistResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryAllowlistResource)(nil)
)

func NewRepositoryAllowlistResource() resource.Resource { return &repositoryAllowlistResource{} }

type repositoryAllowlistResource struct {
	client *client.Client
}

type repositoryAllowlistResourceModel struct {
	ID            types.String `tfsdk:"id"`
	RepositoryKey types.String `tfsdk:"repository_key"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Entries       types.List   `tfsdk:"entries"`
}

type allowlistEntryModel struct {
	Name    types.String `tfsdk:"name"`
	Version types.String `tfsdk:"version"`
	Subdirs types.List   `tfsdk:"subdirs"`
}

var allowlistEntryAttrTypes = map[string]attr.Type{
	"name":    types.StringType,
	"version": types.StringType,
	"subdirs": types.ListType{ElemType: types.StringType},
}

func (r *repositoryAllowlistResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_allowlist"
}

func (r *repositoryAllowlistResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The package allowlist of a virtual conda channel (Artifact Keeper 1.11.0): while enabled, a package from a Remote member is served only if an entry admits it; hosted members are not filtered. A singleton keyed by repository key; destroy removes the list.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Equal to `repository_key`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the virtual conda repository. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the list is enforced. An enabled empty list admits nothing from Remote members.",
			},
			"entries": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Admitted packages.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":    schema.StringAttribute{Required: true, MarkdownDescription: "Exact conda package name or a glob (`*`, `?`), case-insensitive."},
						"version": schema.StringAttribute{Optional: true, MarkdownDescription: "Conda version spec (`2.2.3`, `2.2.*`, `>=2,<3`). Omit for every version."},
						"subdirs": schema.ListAttribute{ElementType: types.StringType, Optional: true, MarkdownDescription: "Subdirs the entry applies to (`noarch`, `linux-64`, ...). Omit for every subdir."},
					},
				},
			},
		},
	}
}

func (r *repositoryAllowlistResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositoryAllowlistResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryAllowlistResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.put(ctx, plan, &resp.State, &resp.Diagnostics)
}

func (r *repositoryAllowlistResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryAllowlistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	a, err := r.client.GetRepositoryAllowlist(ctx, state.RepositoryKey.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading allowlist", err.Error())
		return
	}
	m, d := allowlistToModel(ctx, state.RepositoryKey.ValueString(), a)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *repositoryAllowlistResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryAllowlistResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.put(ctx, plan, &resp.State, &resp.Diagnostics)
}

func (r *repositoryAllowlistResource) put(ctx context.Context, plan repositoryAllowlistResourceModel, state *tfsdk.State, diags *diag.Diagnostics) {
	body := client.RepositoryAllowlist{Enabled: plan.Enabled.ValueBool(), Entries: []client.AllowlistEntry{}}
	if !plan.Entries.IsNull() && !plan.Entries.IsUnknown() {
		var entries []allowlistEntryModel
		diags.Append(plan.Entries.ElementsAs(ctx, &entries, false)...)
		for _, e := range entries {
			subdirs, d := listToStringSlice(ctx, e.Subdirs)
			diags.Append(d...)
			body.Entries = append(body.Entries, client.AllowlistEntry{
				Name:    e.Name.ValueString(),
				Version: optionalString(e.Version),
				Subdirs: subdirs,
			})
		}
	}
	if diags.HasError() {
		return
	}
	key := plan.RepositoryKey.ValueString()
	a, err := r.client.SetRepositoryAllowlist(ctx, key, body)
	if err != nil {
		diags.AddError("Error saving allowlist", err.Error())
		return
	}
	m, d := allowlistToModel(ctx, key, a)
	diags.Append(d...)
	diags.Append(state.Set(ctx, m)...)
}

func (r *repositoryAllowlistResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryAllowlistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRepositoryAllowlist(ctx, state.RepositoryKey.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting allowlist", err.Error())
	}
}

func (r *repositoryAllowlistResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

func allowlistToModel(ctx context.Context, key string, a *client.RepositoryAllowlistResponse) (repositoryAllowlistResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	m := repositoryAllowlistResourceModel{
		ID:            types.StringValue(key),
		RepositoryKey: types.StringValue(key),
		Enabled:       types.BoolValue(a.Enabled),
		Entries:       types.ListNull(types.ObjectType{AttrTypes: allowlistEntryAttrTypes}),
	}
	if len(a.Entries) == 0 {
		return m, diags
	}
	entries := make([]allowlistEntryModel, 0, len(a.Entries))
	for _, e := range a.Entries {
		subdirs, d := stringListValue(ctx, nonEmpty(e.Subdirs))
		diags.Append(d...)
		entries = append(entries, allowlistEntryModel{
			Name:    types.StringValue(e.Name),
			Version: stringPointerValue(e.Version),
			Subdirs: subdirs,
		})
	}
	l, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: allowlistEntryAttrTypes}, entries)
	diags.Append(d...)
	m.Entries = l
	return m, diags
}
