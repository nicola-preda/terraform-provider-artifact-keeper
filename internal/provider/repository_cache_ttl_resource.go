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

var (
	_ resource.Resource                = (*repositoryCacheTtlResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryCacheTtlResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryCacheTtlResource)(nil)
)

func NewRepositoryCacheTtlResource() resource.Resource {
	return &repositoryCacheTtlResource{}
}

type repositoryCacheTtlResource struct {
	client *client.Client
}

type repositoryCacheTtlResourceModel struct {
	ID              types.String `tfsdk:"id"`
	RepositoryKey   types.String `tfsdk:"repository_key"`
	CacheTtlSeconds types.Int64  `tfsdk:"cache_ttl_seconds"`
}

func (r *repositoryCacheTtlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_cache_ttl"
}

func (r *repositoryCacheTtlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Per-repository cache TTL: how long, in seconds, cached upstream artifacts stay fresh before a remote repository re-fetches them. A singleton keyed by repository key: no create/delete (the value lives with the repository), and writes upsert.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to `repository_key`, since the config is a singleton per repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the repository whose cache TTL is managed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cache_ttl_seconds": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "How long, in seconds, cached artifacts stay fresh before being re-fetched from upstream.",
			},
		},
	}
}

func (r *repositoryCacheTtlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositoryCacheTtlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryCacheTtlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	if err := r.client.SetRepositoryCacheTtl(ctx, repoKey, cacheTtlRequestFromModel(plan)); err != nil {
		resp.Diagnostics.AddError("Error configuring repository cache TTL", err.Error())
		return
	}
	cfg, err := r.client.GetRepositoryCacheTtl(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository cache TTL", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, cacheTtlToModel(repoKey, cfg))...)
}

func (r *repositoryCacheTtlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryCacheTtlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := state.RepositoryKey.ValueString()
	cfg, err := r.client.GetRepositoryCacheTtl(ctx, repoKey)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository cache TTL", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, cacheTtlToModel(repoKey, cfg))...)
}

func (r *repositoryCacheTtlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryCacheTtlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	if err := r.client.SetRepositoryCacheTtl(ctx, repoKey, cacheTtlRequestFromModel(plan)); err != nil {
		resp.Diagnostics.AddError("Error updating repository cache TTL", err.Error())
		return
	}
	cfg, err := r.client.GetRepositoryCacheTtl(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository cache TTL", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, cacheTtlToModel(repoKey, cfg))...)
}

// No-op: no delete/reset endpoint. The TTL lives with the repository; destroy
// just drops it from state.
func (r *repositoryCacheTtlResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *repositoryCacheTtlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Addressed by repository key in the API; import by that key.
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

func cacheTtlRequestFromModel(m repositoryCacheTtlResourceModel) client.SetCacheTtlRequest {
	return client.SetCacheTtlRequest{
		CacheTtlSeconds: m.CacheTtlSeconds.ValueInt64(),
	}
}

func cacheTtlToModel(repoKey string, c *client.CacheTtlResponse) repositoryCacheTtlResourceModel {
	return repositoryCacheTtlResourceModel{
		ID:              types.StringValue(repoKey),
		RepositoryKey:   types.StringValue(repoKey),
		CacheTtlSeconds: types.Int64Value(c.CacheTtlSeconds),
	}
}
