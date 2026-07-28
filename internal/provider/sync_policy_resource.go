package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*syncPolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*syncPolicyResource)(nil)
	_ resource.ResourceWithImportState = (*syncPolicyResource)(nil)
)

func NewSyncPolicyResource() resource.Resource { return &syncPolicyResource{} }

type syncPolicyResource struct {
	client *client.Client
}

// syncPolicyResourceModel maps the resource schema. Attribute names mirror the
// Artifact Keeper API fields exactly.
type syncPolicyResourceModel struct {
	ID              types.String         `tfsdk:"id"`
	Name            types.String         `tfsdk:"name"`
	Description     types.String         `tfsdk:"description"`
	Enabled         types.Bool           `tfsdk:"enabled"`
	RepoSelector    *repoSelectorModel   `tfsdk:"repo_selector"`
	PeerSelector    *peerSelectorModel   `tfsdk:"peer_selector"`
	ReplicationMode types.String         `tfsdk:"replication_mode"`
	Priority        types.Int64          `tfsdk:"priority"`
	ArtifactFilter  *artifactFilterModel `tfsdk:"artifact_filter"`
	Filter          types.String         `tfsdk:"filter"`
	Precedence      types.Int64          `tfsdk:"precedence"`
	CreatedAt       types.String         `tfsdk:"created_at"`
	UpdatedAt       types.String         `tfsdk:"updated_at"`
}

type repoSelectorModel struct {
	MatchLabels  types.Map    `tfsdk:"match_labels"`
	MatchFormats types.List   `tfsdk:"match_formats"`
	MatchPattern types.String `tfsdk:"match_pattern"`
	MatchRepos   types.List   `tfsdk:"match_repos"`
}

type peerSelectorModel struct {
	All         types.Bool   `tfsdk:"all"`
	MatchLabels types.Map    `tfsdk:"match_labels"`
	MatchRegion types.String `tfsdk:"match_region"`
	MatchPeers  types.List   `tfsdk:"match_peers"`
}

type artifactFilterModel struct {
	MaxAgeDays   types.Int64 `tfsdk:"max_age_days"`
	IncludePaths types.List  `tfsdk:"include_paths"`
	ExcludePaths types.List  `tfsdk:"exclude_paths"`
	MaxSizeBytes types.Int64 `tfsdk:"max_size_bytes"`
	MatchTags    types.Map   `tfsdk:"match_tags"`
}

func (r *syncPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sync_policy"
}

func (r *syncPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a peer replication sync policy: declarative selectors that map repositories to peer instances for replication. Sync policies are a global federation feature and are admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Sync policy UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique sync policy name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Free-form description. Defaults to an empty string.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the policy is active. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"repo_selector": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Selects which repositories the policy replicates. All active criteria combine with AND semantics. Omit for a policy that matches no repositories. Provide at least one criterion when set.",
				Attributes: map[string]schema.Attribute{
					"match_labels": schema.MapAttribute{
						ElementType: types.StringType, Optional: true,
						MarkdownDescription: "Repository label key/values that must all match.",
					},
					"match_formats": schema.ListAttribute{
						ElementType: types.StringType, Optional: true,
						MarkdownDescription: "Repository formats to include (e.g. `docker`, `maven`). OR semantics.",
					},
					"match_pattern": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Glob-like repository key pattern (only `*` is supported).",
					},
					"match_repos": schema.ListAttribute{
						ElementType: types.StringType, Optional: true,
						MarkdownDescription: "Explicit repository UUIDs to include.",
					},
				},
			},
			"peer_selector": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Selects which peer instances the policy replicates to. Omit for a policy that matches no peers. Provide at least one criterion when set.",
				Attributes: map[string]schema.Attribute{
					"all": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Match all non-local peer instances.",
					},
					"match_labels": schema.MapAttribute{
						ElementType: types.StringType, Optional: true,
						MarkdownDescription: "Peer label key/values that must all match.",
					},
					"match_region": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Match peers in a specific region.",
					},
					"match_peers": schema.ListAttribute{
						ElementType: types.StringType, Optional: true,
						MarkdownDescription: "Explicit peer instance UUIDs to include.",
					},
				},
			},
			"replication_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Replication mode: `push`, `pull`, or `mirror`. Defaults to `push`.",
				Validators:          []validator.String{stringvalidator.OneOf("push", "pull", "mirror")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"priority": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Policy priority. Defaults to `0`.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"artifact_filter": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Constrains which artifacts are eligible for replication. Omit to replicate everything. Provide at least one criterion when set.",
				Attributes: map[string]schema.Attribute{
					"max_age_days": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Only sync artifacts created within the last N days.",
					},
					"include_paths": schema.ListAttribute{
						ElementType: types.StringType, Optional: true,
						MarkdownDescription: "Glob patterns for artifact paths to include. Mirrored by the computed `filter` shorthand.",
					},
					"exclude_paths": schema.ListAttribute{
						ElementType: types.StringType, Optional: true,
						MarkdownDescription: "Glob patterns for artifact paths to exclude.",
					},
					"max_size_bytes": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Maximum artifact size in bytes.",
					},
					"match_tags": schema.MapAttribute{
						ElementType: types.StringType, Optional: true,
						MarkdownDescription: "Tag key/values that must all match (empty value means the key must exist with any value).",
					},
				},
			},
			"filter": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Convenience glob mirrored from `artifact_filter.include_paths` (first entry, or empty). Set include paths via `artifact_filter`; this attribute is read-only and recomputed on each apply.",
			},
			"precedence": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Policy precedence; lower is evaluated first. Defaults to `100`.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 last-update timestamp.",
			},
		},
	}
}

func (r *syncPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *syncPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan syncPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, d := syncPolicyRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.CreateSyncPolicy(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating sync policy", err.Error())
		return
	}

	state, d := syncPolicyToModel(ctx, policy)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *syncPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state syncPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetSyncPolicy(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading sync policy", err.Error())
		return
	}

	refreshed, d := syncPolicyToModel(ctx, policy)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *syncPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan syncPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, d := syncPolicyRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.UpdateSyncPolicy(ctx, plan.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating sync policy", err.Error())
		return
	}

	state, d := syncPolicyToModel(ctx, policy)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *syncPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state syncPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSyncPolicy(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting sync policy", err.Error())
	}
}

func (r *syncPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// syncPolicyRequestFromModel builds the create/update payload. Selectors are
// always sent (empty when unset) so an update can clear them; scalar fields
// with server defaults are sent only when known.
func syncPolicyRequestFromModel(ctx context.Context, m syncPolicyResourceModel) (client.SyncPolicyRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	repoSel, d := buildRepoSelector(ctx, m.RepoSelector)
	diags.Append(d...)
	peerSel, d := buildPeerSelector(ctx, m.PeerSelector)
	diags.Append(d...)
	artFilter, d := buildArtifactFilter(ctx, m.ArtifactFilter)
	diags.Append(d...)

	req := client.SyncPolicyRequest{
		Name:           m.Name.ValueString(),
		RepoSelector:   repoSel,
		PeerSelector:   peerSel,
		ArtifactFilter: artFilter,
	}
	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		req.Description = m.Description.ValueStringPointer()
	}
	if !m.Enabled.IsNull() && !m.Enabled.IsUnknown() {
		req.Enabled = m.Enabled.ValueBoolPointer()
	}
	if !m.ReplicationMode.IsNull() && !m.ReplicationMode.IsUnknown() {
		req.ReplicationMode = m.ReplicationMode.ValueStringPointer()
	}
	if !m.Priority.IsNull() && !m.Priority.IsUnknown() {
		req.Priority = m.Priority.ValueInt64Pointer()
	}
	if !m.Precedence.IsNull() && !m.Precedence.IsUnknown() {
		req.Precedence = m.Precedence.ValueInt64Pointer()
	}
	return req, diags
}

func buildRepoSelector(ctx context.Context, m *repoSelectorModel) (*client.RepoSelector, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &client.RepoSelector{}
	if m == nil {
		return out, diags
	}
	labels, d := mapToStringMap(ctx, m.MatchLabels)
	diags.Append(d...)
	formats, d := listToStringSlice(ctx, m.MatchFormats)
	diags.Append(d...)
	repos, d := listToStringSlice(ctx, m.MatchRepos)
	diags.Append(d...)
	out.MatchLabels = labels
	out.MatchFormats = formats
	out.MatchPattern = optionalString(m.MatchPattern)
	out.MatchRepos = repos
	return out, diags
}

func buildPeerSelector(ctx context.Context, m *peerSelectorModel) (*client.PeerSelector, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &client.PeerSelector{}
	if m == nil {
		return out, diags
	}
	labels, d := mapToStringMap(ctx, m.MatchLabels)
	diags.Append(d...)
	peers, d := listToStringSlice(ctx, m.MatchPeers)
	diags.Append(d...)
	if !m.All.IsNull() && !m.All.IsUnknown() {
		out.All = m.All.ValueBool()
	}
	out.MatchLabels = labels
	out.MatchRegion = optionalString(m.MatchRegion)
	out.MatchPeers = peers
	return out, diags
}

func buildArtifactFilter(ctx context.Context, m *artifactFilterModel) (*client.ArtifactFilter, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &client.ArtifactFilter{}
	if m == nil {
		return out, diags
	}
	include, d := listToStringSlice(ctx, m.IncludePaths)
	diags.Append(d...)
	exclude, d := listToStringSlice(ctx, m.ExcludePaths)
	diags.Append(d...)
	tags, d := mapToStringMap(ctx, m.MatchTags)
	diags.Append(d...)
	if !m.MaxAgeDays.IsNull() && !m.MaxAgeDays.IsUnknown() {
		out.MaxAgeDays = m.MaxAgeDays.ValueInt64Pointer()
	}
	if !m.MaxSizeBytes.IsNull() && !m.MaxSizeBytes.IsUnknown() {
		out.MaxSizeBytes = m.MaxSizeBytes.ValueInt64Pointer()
	}
	out.IncludePaths = include
	out.ExcludePaths = exclude
	out.MatchTags = tags
	return out, diags
}

func syncPolicyToModel(ctx context.Context, p *client.SyncPolicy) (syncPolicyResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	repoSel, d := repoSelectorToModel(ctx, p.RepoSelector)
	diags.Append(d...)
	peerSel, d := peerSelectorToModel(ctx, p.PeerSelector)
	diags.Append(d...)
	artFilter, d := artifactFilterToModel(ctx, p.ArtifactFilter)
	diags.Append(d...)

	return syncPolicyResourceModel{
		ID:              types.StringValue(p.ID),
		Name:            types.StringValue(p.Name),
		Description:     types.StringValue(p.Description),
		Enabled:         types.BoolValue(p.Enabled),
		RepoSelector:    repoSel,
		PeerSelector:    peerSel,
		ReplicationMode: types.StringValue(p.ReplicationMode),
		Priority:        types.Int64Value(p.Priority),
		ArtifactFilter:  artFilter,
		Filter:          types.StringValue(p.Filter),
		Precedence:      types.Int64Value(p.Precedence),
		CreatedAt:       types.StringValue(p.CreatedAt),
		UpdatedAt:       types.StringValue(p.UpdatedAt),
	}, diags
}

// repoSelectorToModel maps a decoded selector back to nested state. The API
// echoes fully-populated selectors, so empty collections/patterns are folded
// to null and a wholly-empty selector becomes a null object to match config.
func repoSelectorToModel(ctx context.Context, s client.RepoSelector) (*repoSelectorModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	labels := s.MatchLabels
	if len(labels) == 0 {
		labels = nil
	}
	formats := s.MatchFormats
	if len(formats) == 0 {
		formats = nil
	}
	repos := s.MatchRepos
	if len(repos) == 0 {
		repos = nil
	}
	if labels == nil && formats == nil && repos == nil && s.MatchPattern == nil {
		return nil, diags
	}
	labelsVal, d := stringMapValue(ctx, labels)
	diags.Append(d...)
	formatsVal, d := stringListValue(ctx, formats)
	diags.Append(d...)
	reposVal, d := stringListValue(ctx, repos)
	diags.Append(d...)
	return &repoSelectorModel{
		MatchLabels:  labelsVal,
		MatchFormats: formatsVal,
		MatchPattern: stringPointerValue(s.MatchPattern),
		MatchRepos:   reposVal,
	}, diags
}

func peerSelectorToModel(ctx context.Context, s client.PeerSelector) (*peerSelectorModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	labels := s.MatchLabels
	if len(labels) == 0 {
		labels = nil
	}
	peers := s.MatchPeers
	if len(peers) == 0 {
		peers = nil
	}
	if !s.All && labels == nil && peers == nil && s.MatchRegion == nil {
		return nil, diags
	}
	labelsVal, d := stringMapValue(ctx, labels)
	diags.Append(d...)
	peersVal, d := stringListValue(ctx, peers)
	diags.Append(d...)
	return &peerSelectorModel{
		All:         types.BoolValue(s.All),
		MatchLabels: labelsVal,
		MatchRegion: stringPointerValue(s.MatchRegion),
		MatchPeers:  peersVal,
	}, diags
}

func artifactFilterToModel(ctx context.Context, f client.ArtifactFilter) (*artifactFilterModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	include := f.IncludePaths
	if len(include) == 0 {
		include = nil
	}
	exclude := f.ExcludePaths
	if len(exclude) == 0 {
		exclude = nil
	}
	tags := f.MatchTags
	if len(tags) == 0 {
		tags = nil
	}
	if include == nil && exclude == nil && tags == nil && f.MaxAgeDays == nil && f.MaxSizeBytes == nil {
		return nil, diags
	}
	includeVal, d := stringListValue(ctx, include)
	diags.Append(d...)
	excludeVal, d := stringListValue(ctx, exclude)
	diags.Append(d...)
	tagsVal, d := stringMapValue(ctx, tags)
	diags.Append(d...)
	return &artifactFilterModel{
		MaxAgeDays:   int64PointerValue(f.MaxAgeDays),
		IncludePaths: includeVal,
		ExcludePaths: excludeVal,
		MaxSizeBytes: int64PointerValue(f.MaxSizeBytes),
		MatchTags:    tagsVal,
	}, diags
}
