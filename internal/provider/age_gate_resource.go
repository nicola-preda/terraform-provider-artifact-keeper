package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	_ resource.Resource                = (*ageGateResource)(nil)
	_ resource.ResourceWithConfigure   = (*ageGateResource)(nil)
	_ resource.ResourceWithImportState = (*ageGateResource)(nil)
)

func NewAgeGateResource() resource.Resource { return &ageGateResource{} }

type ageGateResource struct {
	client *client.Client
}

type ageGateResourceModel struct {
	RepositoryKey types.String `tfsdk:"repository_key"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	MinAgeDays    types.Int64  `tfsdk:"min_age_days"`
}

func (r *ageGateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_age_gate"
}

func (r *ageGateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Age-based proxy gate for a single `remote` repository: holds newly published upstream versions until they reach a minimum age. A per-repository singleton keyed by repository key (no create/delete); applies only to remote repositories.",
		Attributes: map[string]schema.Attribute{
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the remote repository whose age gate is configured. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the age gate is enforced for this repository.",
			},
			"min_age_days": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Minimum age, in days, an upstream version must reach before it is served. `0` is the trusted-remote setting (no delay, but explicit rejections still block). Must be between 0 and 3650.",
				Validators:          []validator.Int64{int64validator.Between(0, 3650)},
			},
		},
	}
}

func (r *ageGateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *ageGateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ageGateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No create endpoint: the gate config always exists on a remote repo. The
	// PUT establishes the managed configuration.
	cfg, err := r.client.UpdateAgeGateConfig(ctx, plan.RepositoryKey.ValueString(), client.AgeGateConfigRequest{
		Enabled:    plan.Enabled.ValueBool(),
		MinAgeDays: plan.MinAgeDays.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error configuring age gate", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ageGateToModel(cfg))...)
}

func (r *ageGateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ageGateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetAgeGateConfig(ctx, state.RepositoryKey.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading age gate", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ageGateToModel(cfg))...)
}

func (r *ageGateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ageGateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.UpdateAgeGateConfig(ctx, plan.RepositoryKey.ValueString(), client.AgeGateConfigRequest{
		Enabled:    plan.Enabled.ValueBool(),
		MinAgeDays: plan.MinAgeDays.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating age gate", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ageGateToModel(cfg))...)
}

func (r *ageGateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ageGateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// There is no delete/reset endpoint; destroying the resource disables the
	// gate by PUTing the disabled default (enabled=false, min_age_days=0).
	// A 404 means the repository is already gone, which is fine.
	if _, err := r.client.UpdateAgeGateConfig(ctx, state.RepositoryKey.ValueString(), client.AgeGateConfigRequest{
		Enabled:    false,
		MinAgeDays: 0,
	}); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error disabling age gate", err.Error())
	}
}

func (r *ageGateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Addressed by repository key in the API; import by key.
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

func ageGateToModel(c *client.AgeGateConfig) ageGateResourceModel {
	return ageGateResourceModel{
		RepositoryKey: types.StringValue(c.RepositoryKey),
		Enabled:       types.BoolValue(c.Enabled),
		MinAgeDays:    types.Int64Value(c.MinAgeDays),
	}
}
