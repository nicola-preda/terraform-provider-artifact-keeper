package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*repositoryPypiTrackResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryPypiTrackResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryPypiTrackResource)(nil)
)

// pypiNameSep matches a run of PEP 503 separators; normalizePypiProject collapses
// each run to a single dash.
var pypiNameSep = regexp.MustCompile(`[-_.]+`)

// normalizePypiProject applies PEP 503 name normalization: lowercase, then
// collapse runs of `-`, `_`, `.` into a single `-`. Mirrors what the backend
// does to {project}, so we can match a track in the list by normalized name.
func normalizePypiProject(s string) string {
	return pypiNameSep.ReplaceAllString(strings.ToLower(s), "-")
}

func NewRepositoryPypiTrackResource() resource.Resource {
	return &repositoryPypiTrackResource{}
}

type repositoryPypiTrackResource struct {
	client *client.Client
}

// A single tracked project on a hosted PyPI repo; synthetic id is
// "<repository_key>/<project>".
type repositoryPypiTrackResourceModel struct {
	ID             types.String `tfsdk:"id"`
	RepositoryKey  types.String `tfsdk:"repository_key"`
	Project        types.String `tfsdk:"project"`
	TracksURL      types.String `tfsdk:"tracks_url"`
	NormalizedName types.String `tfsdk:"normalized_name"`
}

func (r *repositoryPypiTrackResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_pypi_track"
}

func (r *repositoryPypiTrackResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A PyPI track: a local project on a hosted PyPI repository that mirrors an upstream Simple index. One resource per tracked project. The repository key and project name form the identity, so changing either forces a new resource; only the upstream URL can be updated in place. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synthetic ID in the form `\"<repository_key>/<project>\"`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the hosted PyPI repository that owns the project. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Local project name (a path segment). The backend PEP 503-normalizes it into `normalized_name`. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tracks_url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Upstream Simple index URL the project tracks, e.g. `https://pypi.org/simple/acme-sdk/`.",
			},
			"normalized_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "PEP 503-normalized form of `project` (lowercased, with runs of `-`, `_`, `.` collapsed to a single `-`), as computed by the backend.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *repositoryPypiTrackResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositoryPypiTrackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryPypiTrackResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	project := plan.Project.ValueString()
	if err := r.client.SetPypiTrack(ctx, repoKey, project, client.PypiTrackRequest{TracksURL: plan.TracksURL.ValueString()}); err != nil {
		resp.Diagnostics.AddError("Error creating pypi track", err.Error())
		return
	}

	// No single-track GET; read back from the list by normalized project name.
	track, err := r.client.GetPypiTrack(ctx, repoKey, normalizePypiProject(project))
	if err != nil {
		resp.Diagnostics.AddError("Error reading pypi track", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, pypiTrackToModel(repoKey, project, track))...)
}

func (r *repositoryPypiTrackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryPypiTrackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := state.RepositoryKey.ValueString()
	project := state.Project.ValueString()
	track, err := r.client.GetPypiTrack(ctx, repoKey, normalizePypiProject(project))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading pypi track", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, pypiTrackToModel(repoKey, project, track))...)
}

func (r *repositoryPypiTrackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryPypiTrackResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only tracks_url can change here; repository_key and project force replacement.
	repoKey := plan.RepositoryKey.ValueString()
	project := plan.Project.ValueString()
	if err := r.client.SetPypiTrack(ctx, repoKey, project, client.PypiTrackRequest{TracksURL: plan.TracksURL.ValueString()}); err != nil {
		resp.Diagnostics.AddError("Error updating pypi track", err.Error())
		return
	}

	track, err := r.client.GetPypiTrack(ctx, repoKey, normalizePypiProject(project))
	if err != nil {
		resp.Diagnostics.AddError("Error reading pypi track", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, pypiTrackToModel(repoKey, project, track))...)
}

func (r *repositoryPypiTrackResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryPypiTrackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePypiTrack(ctx, state.RepositoryKey.ValueString(), state.Project.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting pypi track", err.Error())
	}
}

// Composite import ID "<repository_key>/<project>", split on the last "/" so a
// project name never eats into the key.
func (r *repositoryPypiTrackResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	i := strings.LastIndex(req.ID, "/")
	if i <= 0 || i == len(req.ID)-1 {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected import ID in the format \"<repository_key>/<project>\", got: %q", req.ID),
		)
		return
	}
	repoKey, project := req.ID[:i], req.ID[i+1:]
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository_key"), repoKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func pypiTrackToModel(repoKey, project string, t *client.PypiTrackResponse) repositoryPypiTrackResourceModel {
	return repositoryPypiTrackResourceModel{
		ID:             types.StringValue(repoKey + "/" + project),
		RepositoryKey:  types.StringValue(repoKey),
		Project:        types.StringValue(project),
		TracksURL:      types.StringValue(t.TracksURL),
		NormalizedName: types.StringValue(t.NormalizedName),
	}
}
