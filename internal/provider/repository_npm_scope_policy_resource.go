package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*repositoryNpmScopePolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryNpmScopePolicyResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryNpmScopePolicyResource)(nil)
)

func NewRepositoryNpmScopePolicyResource() resource.Resource {
	return &repositoryNpmScopePolicyResource{}
}

type repositoryNpmScopePolicyResource struct {
	client *client.Client
}

type repositoryNpmScopePolicyResourceModel struct {
	ID            types.String `tfsdk:"id"`
	RepositoryKey types.String `tfsdk:"repository_key"`
	AllowedScopes types.List   `tfsdk:"allowed_scopes"`
	AllowUnscoped types.Bool   `tfsdk:"allow_unscoped"`
	Active        types.Bool   `tfsdk:"active"`
}

func (r *repositoryNpmScopePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_npm_scope_policy"
}

func (r *repositoryNpmScopePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The npm scope allowlist for a repository: which `@scope` packages it may serve and whether unscoped packages are allowed. A singleton keyed by repository key: no create/delete (the policy lives with the repository), and writes replace the stored policy. Only meaningful for a remote member of an npm virtual repository; the backend rejects it otherwise.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to `repository_key`, since the policy is a singleton per repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the repository whose npm scope policy is managed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"allowed_scopes": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "npm scopes the repository may serve, each starting with `@` (e.g. `@myorg`). Omit or leave empty to impose no scope restriction.",
			},
			"allow_unscoped": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether unscoped packages (those without an `@scope`) are allowed. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the stored policy actually restricts anything (i.e. it is enforced rather than a no-op).",
			},
		},
	}
}

func (r *repositoryNpmScopePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositoryNpmScopePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryNpmScopePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	apiReq, d := npmScopePolicyRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetNpmScopePolicy(ctx, repoKey, apiReq); err != nil {
		resp.Diagnostics.AddError("Error configuring repository npm scope policy", err.Error())
		return
	}
	policy, err := r.client.GetNpmScopePolicy(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository npm scope policy", err.Error())
		return
	}

	state, d := npmScopePolicyToModel(ctx, repoKey, policy)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *repositoryNpmScopePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryNpmScopePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := state.RepositoryKey.ValueString()
	policy, err := r.client.GetNpmScopePolicy(ctx, repoKey)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository npm scope policy", err.Error())
		return
	}

	refreshed, d := npmScopePolicyToModel(ctx, repoKey, policy)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *repositoryNpmScopePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryNpmScopePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	apiReq, d := npmScopePolicyRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetNpmScopePolicy(ctx, repoKey, apiReq); err != nil {
		resp.Diagnostics.AddError("Error updating repository npm scope policy", err.Error())
		return
	}
	policy, err := r.client.GetNpmScopePolicy(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository npm scope policy", err.Error())
		return
	}

	state, d := npmScopePolicyToModel(ctx, repoKey, policy)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// No-op: no delete endpoint. The policy lives with the repository; destroy just
// drops it from state.
func (r *repositoryNpmScopePolicyResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *repositoryNpmScopePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Addressed by repository key in the API; import by that key.
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

func npmScopePolicyRequestFromModel(ctx context.Context, m repositoryNpmScopePolicyResourceModel) (client.SetNpmScopePolicyRequest, diag.Diagnostics) {
	scopes, diags := listToStringSlice(ctx, m.AllowedScopes)
	// The PUT replaces the policy, so always send the list; a null list means
	// "no restriction", which the backend takes as an empty allowlist.
	if scopes == nil {
		scopes = []string{}
	}
	return client.SetNpmScopePolicyRequest{
		AllowedScopes: scopes,
		AllowUnscoped: m.AllowUnscoped.ValueBool(),
	}, diags
}

func npmScopePolicyToModel(ctx context.Context, repoKey string, p *client.NpmScopePolicy) (repositoryNpmScopePolicyResourceModel, diag.Diagnostics) {
	// An empty allowlist means "no restriction", the same as omitting it, so keep
	// the attribute null to let an unset config round-trip cleanly.
	var scopeVals []string
	if len(p.AllowedScopes) > 0 {
		scopeVals = p.AllowedScopes
	}
	scopes, diags := stringListValue(ctx, scopeVals)
	return repositoryNpmScopePolicyResourceModel{
		ID:            types.StringValue(repoKey),
		RepositoryKey: types.StringValue(repoKey),
		AllowedScopes: scopes,
		AllowUnscoped: types.BoolValue(p.AllowUnscoped),
		Active:        types.BoolValue(p.Active),
	}, diags
}
