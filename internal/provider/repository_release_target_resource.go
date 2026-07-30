package provider

import (
	"context"

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
	_ resource.Resource                = (*repositoryReleaseTargetResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryReleaseTargetResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryReleaseTargetResource)(nil)
)

func NewRepositoryReleaseTargetResource() resource.Resource {
	return &repositoryReleaseTargetResource{}
}

type repositoryReleaseTargetResource struct {
	client *client.Client
}

type repositoryReleaseTargetResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	RepositoryKey        types.String `tfsdk:"repository_key"`
	ReleaseRepositoryKey types.String `tfsdk:"release_repository_key"`
	Linked               types.Bool   `tfsdk:"linked"`
	ReleaseRepositoryID  types.String `tfsdk:"release_repository_id"`
}

func (r *repositoryReleaseTargetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_release_target"
}

func (r *repositoryReleaseTargetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Promotion release target of a staging repository: the release repository that artifacts promote into. A singleton keyed by staging repository key, so no create/delete of its own (the link lives with the repository), and writes upsert. Omit `release_repository_key` to leave the staging repo unlinked. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to `repository_key`, since the release target is a singleton per staging repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the staging repository whose release target is managed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"release_repository_key": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Key of the release repository to link as the promotion target. Omit to leave the staging repo unlinked.",
			},
			"linked": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the staging repository is linked to a release repository.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"release_repository_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the linked release repository (returned by the API), null when unlinked.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *repositoryReleaseTargetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositoryReleaseTargetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryReleaseTargetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	if err := r.client.SetReleaseTarget(ctx, repoKey, releaseTargetRequestFromModel(plan)); err != nil {
		resp.Diagnostics.AddError("Error configuring repository release target", err.Error())
		return
	}
	target, err := r.client.GetReleaseTarget(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository release target", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, releaseTargetToModel(repoKey, target))...)
}

func (r *repositoryReleaseTargetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryReleaseTargetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := state.RepositoryKey.ValueString()
	target, err := r.client.GetReleaseTarget(ctx, repoKey)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository release target", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, releaseTargetToModel(repoKey, target))...)
}

func (r *repositoryReleaseTargetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryReleaseTargetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	if err := r.client.SetReleaseTarget(ctx, repoKey, releaseTargetRequestFromModel(plan)); err != nil {
		resp.Diagnostics.AddError("Error updating repository release target", err.Error())
		return
	}
	target, err := r.client.GetReleaseTarget(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository release target", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, releaseTargetToModel(repoKey, target))...)
}

// Delete unlinks the staging repo (PUT with a nil release key). There is no
// delete endpoint: the target lives with the repository.
func (r *repositoryReleaseTargetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryReleaseTargetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetReleaseTarget(ctx, state.RepositoryKey.ValueString(), client.SetReleaseTargetRequest{ReleaseRepositoryKey: nil}); err != nil {
		resp.Diagnostics.AddError("Error unlinking repository release target", err.Error())
		return
	}
}

func (r *repositoryReleaseTargetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Addressed by staging repository key in the API; import by that key.
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

func releaseTargetRequestFromModel(m repositoryReleaseTargetResourceModel) client.SetReleaseTargetRequest {
	return client.SetReleaseTargetRequest{
		ReleaseRepositoryKey: optionalString(m.ReleaseRepositoryKey),
	}
}

func releaseTargetToModel(repoKey string, t *client.ReleaseTargetResponse) repositoryReleaseTargetResourceModel {
	return repositoryReleaseTargetResourceModel{
		ID:                   types.StringValue(repoKey),
		RepositoryKey:        types.StringValue(repoKey),
		ReleaseRepositoryKey: stringPointerValue(t.ReleaseRepositoryKey),
		Linked:               types.BoolValue(t.Linked),
		ReleaseRepositoryID:  stringPointerValue(t.ReleaseRepositoryID),
	}
}
