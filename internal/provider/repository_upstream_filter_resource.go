package provider

import (
	"context"

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
	_ resource.Resource                = (*repositoryUpstreamFilterResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryUpstreamFilterResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryUpstreamFilterResource)(nil)
)

func NewRepositoryUpstreamFilterResource() resource.Resource {
	return &repositoryUpstreamFilterResource{}
}

type repositoryUpstreamFilterResource struct {
	client *client.Client
}

type repositoryUpstreamFilterResourceModel struct {
	ID              types.String `tfsdk:"id"`
	RepositoryKey   types.String `tfsdk:"repository_key"`
	IncludePatterns types.List   `tfsdk:"include_patterns"`
	ExcludePatterns types.List   `tfsdk:"exclude_patterns"`
	Active          types.Bool   `tfsdk:"active"`
}

func (r *repositoryUpstreamFilterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_upstream_filter"
}

func (r *repositoryUpstreamFilterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Which paths a Remote repository may request from its upstream (Artifact Keeper 1.11.0). A path outside the filter is answered 404 without contacting the upstream, and a virtual repository skips a member whose filter excludes the path. A singleton keyed by repository key; destroy removes the filter.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Equal to `repository_key`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the Remote repository. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"include_patterns": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "When set, only paths matching at least one of these regexes are fetched (search semantics; anchor with `^`/`$`).",
			},
			"exclude_patterns": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Paths matching any of these regexes are never fetched. Exclusion wins over inclusion.",
			},
			"active": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether a filter is enforced."},
		},
	}
}

func (r *repositoryUpstreamFilterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositoryUpstreamFilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryUpstreamFilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.put(ctx, plan, &resp.State, &resp.Diagnostics)
}

func (r *repositoryUpstreamFilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryUpstreamFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	f, err := r.client.GetUpstreamFilter(ctx, state.RepositoryKey.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading upstream filter", err.Error())
		return
	}
	m, d := upstreamFilterToModel(ctx, state.RepositoryKey.ValueString(), f)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *repositoryUpstreamFilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryUpstreamFilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.put(ctx, plan, &resp.State, &resp.Diagnostics)
}

func (r *repositoryUpstreamFilterResource) put(ctx context.Context, plan repositoryUpstreamFilterResourceModel, state *tfsdk.State, diags *diag.Diagnostics) {
	inc, d := listToStringSlice(ctx, plan.IncludePatterns)
	diags.Append(d...)
	exc, d := listToStringSlice(ctx, plan.ExcludePatterns)
	diags.Append(d...)
	if inc == nil {
		inc = []string{}
	}
	if exc == nil {
		exc = []string{}
	}
	key := plan.RepositoryKey.ValueString()
	f, err := r.client.SetUpstreamFilter(ctx, key, client.UpstreamFilter{IncludePatterns: inc, ExcludePatterns: exc})
	if err != nil {
		diags.AddError("Error saving upstream filter", err.Error())
		return
	}
	m, d := upstreamFilterToModel(ctx, key, f)
	diags.Append(d...)
	diags.Append(state.Set(ctx, m)...)
}

func (r *repositoryUpstreamFilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryUpstreamFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteUpstreamFilter(ctx, state.RepositoryKey.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting upstream filter", err.Error())
	}
}

func (r *repositoryUpstreamFilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

// upstreamFilterToModel folds an empty list back to null so an unset list
// round-trips.
func upstreamFilterToModel(ctx context.Context, key string, f *client.UpstreamFilterResponse) (repositoryUpstreamFilterResourceModel, diag.Diagnostics) {
	inc, diags := stringListValue(ctx, nonEmpty(f.IncludePatterns))
	exc, d := stringListValue(ctx, nonEmpty(f.ExcludePatterns))
	diags.Append(d...)
	return repositoryUpstreamFilterResourceModel{
		ID:              types.StringValue(key),
		RepositoryKey:   types.StringValue(key),
		IncludePatterns: inc,
		ExcludePatterns: exc,
		Active:          types.BoolValue(f.Active),
	}, diags
}

func nonEmpty(v []string) []string {
	if len(v) == 0 {
		return nil
	}
	return v
}
