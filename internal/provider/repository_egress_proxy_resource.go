package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*repositoryEgressProxyResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryEgressProxyResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryEgressProxyResource)(nil)
)

func NewRepositoryEgressProxyResource() resource.Resource {
	return &repositoryEgressProxyResource{}
}

type repositoryEgressProxyResource struct {
	client *client.Client
}

type repositoryEgressProxyResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	RepositoryKey              types.String `tfsdk:"repository_key"`
	Mode                       types.String `tfsdk:"mode"`
	ProxyURL                   types.String `tfsdk:"proxy_url"`
	NoProxy                    types.String `tfsdk:"no_proxy"`
	ProxyCredentialsConfigured types.Bool   `tfsdk:"proxy_credentials_configured"`
}

func (r *repositoryEgressProxyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_egress_proxy"
}

func (r *repositoryEgressProxyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "How one remote repository reaches its upstream: through the process-wide proxy environment, straight out, or through a proxy of its own. A singleton keyed by repository key (no create/delete, writes upsert), remote repositories only, and admin-gated on read as well as write. The per-repository setting overrides `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY`, it does not merge with them. Destroying the resource resets the repository to `inherit`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to `repository_key`, since the config is a singleton per repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the remote repository whose egress proxy is managed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"mode": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "`inherit` to follow the process-wide proxy environment (the default for a repository that has never been configured), `direct` to bypass it and connect straight out, or `explicit` to use this repository's own `proxy_url`.",
				Validators:          []validator.String{stringvalidator.OneOf("inherit", "direct", "explicit")},
			},
			"proxy_url": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Proxy to use, `http://` or `https://` (socks is rejected), optionally carrying `user:pass@` credentials. Required when `mode` is `explicit`, and rejected for the other modes rather than silently dropped. The API returns this with any credentials replaced by `***`, so a credentialed URL is kept from configuration rather than read back.",
			},
			"no_proxy": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Comma-separated hosts, domain suffixes or CIDRs that bypass the proxy for this repository. Only meaningful when `mode` is `explicit`.",
			},
			"proxy_credentials_configured": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the stored proxy URL carries credentials. The credentials themselves are never returned.",
			},
		},
	}
}

func (r *repositoryEgressProxyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositoryEgressProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryEgressProxyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.set(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring repository egress proxy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, egressProxyToModel(plan, cfg))...)
}

func (r *repositoryEgressProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryEgressProxyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetRepositoryEgressProxy(ctx, state.RepositoryKey.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository egress proxy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, egressProxyToModel(state, cfg))...)
}

func (r *repositoryEgressProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryEgressProxyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.set(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error updating repository egress proxy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, egressProxyToModel(plan, cfg))...)
}

// Delete resets the repository to `inherit` rather than no-opping like the other
// per-repository singletons: an egress control left behind after destroy still
// routes traffic, and the stored proxy URL may hold credentials.
func (r *repositoryEgressProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryEgressProxyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.SetRepositoryEgressProxy(ctx, state.RepositoryKey.ValueString(), client.SetEgressProxyRequest{Mode: "inherit"})
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error resetting repository egress proxy", err.Error())
	}
}

func (r *repositoryEgressProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Addressed by repository key. A credentialed proxy_url imports redacted and
	// is replaced on the next apply.
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

// set PUTs the plan. The PUT echoes the stored config, so no follow-up GET.
func (r *repositoryEgressProxyResource) set(ctx context.Context, plan repositoryEgressProxyResourceModel) (*client.EgressProxy, error) {
	return r.client.SetRepositoryEgressProxy(ctx, plan.RepositoryKey.ValueString(), client.SetEgressProxyRequest{
		Mode:     plan.Mode.ValueString(),
		ProxyURL: optionalString(plan.ProxyURL),
		NoProxy:  optionalString(plan.NoProxy),
	})
}

// egressProxyToModel folds the API read-back onto the configured values.
//
// `proxy_url` is kept from configuration whenever there is one, because the
// server's copy is not comparable to it on two counts: credentials come back
// redacted to `***`, and the URL is normalised (`http://host:3128` reads back as
// `http://host:3128/`). Writing either back over a configured, non-computed
// attribute is a "provider produced inconsistent result after apply" error, and
// on refresh it would be permanent phantom drift. Only an import, which has no
// configured value to keep, takes the server's URL.
//
// `no_proxy` gets the same treatment for the same reason: the server trims it.
// So drift on those two strings is not detected. `mode` and
// `proxy_credentials_configured` do round-trip exactly and are the drift signal,
// alongside a 404 when the repository itself is gone.
func egressProxyToModel(configured repositoryEgressProxyResourceModel, c *client.EgressProxy) repositoryEgressProxyResourceModel {
	// Server values first, then let anything actually configured win.
	m := repositoryEgressProxyResourceModel{
		ID:                         types.StringValue(configured.RepositoryKey.ValueString()),
		RepositoryKey:              configured.RepositoryKey,
		Mode:                       types.StringValue(c.Mode),
		ProxyURL:                   stringPointerValue(c.ProxyURL),
		NoProxy:                    stringPointerValue(c.NoProxy),
		ProxyCredentialsConfigured: types.BoolValue(c.ProxyCredentialsConfigured),
	}
	if isSet(configured.ProxyURL) {
		m.ProxyURL = configured.ProxyURL
	}
	if isSet(configured.NoProxy) {
		m.NoProxy = configured.NoProxy
	}
	return m
}

// isSet reports whether a string attribute carries a usable value, i.e. it came
// from configuration or prior state rather than being absent or not yet known.
func isSet(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }
