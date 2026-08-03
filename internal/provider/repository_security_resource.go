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

var (
	_ resource.Resource                = (*repositorySecurityResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositorySecurityResource)(nil)
	_ resource.ResourceWithImportState = (*repositorySecurityResource)(nil)
)

func NewRepositorySecurityResource() resource.Resource {
	return &repositorySecurityResource{}
}

type repositorySecurityResource struct {
	client *client.Client
}

type repositorySecurityResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	RepositoryKey          types.String `tfsdk:"repository_key"`
	RepositoryID           types.String `tfsdk:"repository_id"`
	ScanEnabled            types.Bool   `tfsdk:"scan_enabled"`
	ScanOnUpload           types.Bool   `tfsdk:"scan_on_upload"`
	ScanOnProxy            types.Bool   `tfsdk:"scan_on_proxy"`
	BlockOnPolicyViolation types.Bool   `tfsdk:"block_on_policy_violation"`
	SeverityThreshold      types.String `tfsdk:"severity_threshold"`
	ProxyScanAction        types.String `tfsdk:"proxy_scan_action"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

func (r *repositorySecurityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_security"
}

func (r *repositorySecurityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Per-repository vulnerability-scanning configuration: whether scanning is on, when it runs (on upload, on proxy fetch), and the policy-violation gate. A singleton keyed by repository key: no create/delete (the config is defaulted with the repository and reset when it is removed), and writes upsert. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to `repository_key`, since the config is a singleton per repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the repository whose scan config is managed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repository_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the repository (returned by the API).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"scan_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether vulnerability scanning is enabled for the repository.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"scan_on_upload": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Scan artifacts when they are uploaded (hosted repositories).",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"scan_on_proxy": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Scan artifacts fetched through a proxy (remote repositories).",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"block_on_policy_violation": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Block serving/promotion of artifacts that violate the security policy.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"severity_threshold": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum finding severity that trips the gate: one of `critical`, `high`, `medium`, `low`, `info`.",
				Validators:          []validator.String{stringvalidator.OneOf("critical", "high", "medium", "low", "info")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"proxy_scan_action": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Behavior when an inline proxy scan-on-fetch cannot run: `fail_open` (default, serve the artifact anyway) or `fail_closed` (refuse it).",
				Validators:          []validator.String{stringvalidator.OneOf("fail_open", "fail_closed")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *repositorySecurityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositorySecurityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositorySecurityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	if err := r.client.UpdateRepositorySecurity(ctx, repoKey, securityRequestFromModel(plan)); err != nil {
		resp.Diagnostics.AddError("Error configuring repository security config", err.Error())
		return
	}
	cfg, err := r.client.GetRepositorySecurity(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository security config", err.Error())
		return
	}
	if cfg == nil {
		resp.Diagnostics.AddError("Missing repository security config", "The API returned no scan config for repository "+repoKey+" after upsert.")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, securityToModel(repoKey, cfg))...)
}

func (r *repositorySecurityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositorySecurityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := state.RepositoryKey.ValueString()
	cfg, err := r.client.GetRepositorySecurity(ctx, repoKey)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository security config", err.Error())
		return
	}
	if cfg == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, securityToModel(repoKey, cfg))...)
}

func (r *repositorySecurityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositorySecurityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	if err := r.client.UpdateRepositorySecurity(ctx, repoKey, securityRequestFromModel(plan)); err != nil {
		resp.Diagnostics.AddError("Error updating repository security config", err.Error())
		return
	}
	cfg, err := r.client.GetRepositorySecurity(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository security config", err.Error())
		return
	}
	if cfg == nil {
		resp.Diagnostics.AddError("Missing repository security config", "The API returned no scan config for repository "+repoKey+" after upsert.")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, securityToModel(repoKey, cfg))...)
}

// No-op: no delete/reset endpoint. The config lives with the repository; destroy
// just drops it from state.
func (r *repositorySecurityResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *repositorySecurityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Addressed by repository key in the API; import by that key.
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

func securityRequestFromModel(m repositorySecurityResourceModel) client.UpdateRepositorySecurityRequest {
	return client.UpdateRepositorySecurityRequest{
		ScanEnabled:            optionalBool(m.ScanEnabled),
		ScanOnUpload:           optionalBool(m.ScanOnUpload),
		ScanOnProxy:            optionalBool(m.ScanOnProxy),
		BlockOnPolicyViolation: optionalBool(m.BlockOnPolicyViolation),
		SeverityThreshold:      optionalString(m.SeverityThreshold),
		ProxyScanAction:        optionalString(m.ProxyScanAction),
	}
}

func securityToModel(repoKey string, c *client.RepositorySecurity) repositorySecurityResourceModel {
	return repositorySecurityResourceModel{
		ID:                     types.StringValue(repoKey),
		RepositoryKey:          types.StringValue(repoKey),
		RepositoryID:           types.StringValue(c.RepositoryID),
		ScanEnabled:            types.BoolValue(c.ScanEnabled),
		ScanOnUpload:           types.BoolValue(c.ScanOnUpload),
		ScanOnProxy:            types.BoolValue(c.ScanOnProxy),
		BlockOnPolicyViolation: types.BoolValue(c.BlockOnPolicyViolation),
		SeverityThreshold:      types.StringValue(c.SeverityThreshold),
		ProxyScanAction:        types.StringValue(c.ProxyScanAction),
		CreatedAt:              types.StringValue(c.CreatedAt),
		UpdatedAt:              types.StringValue(c.UpdatedAt),
	}
}
