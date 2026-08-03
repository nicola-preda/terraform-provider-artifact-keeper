package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*repositoryResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryResource)(nil)
)

func NewRepositoryResource() resource.Resource {
	return &repositoryResource{}
}

type repositoryResource struct {
	client *client.Client
}

type repositoryResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Key                       types.String `tfsdk:"key"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	Format                    types.String `tfsdk:"format"`
	RepoType                  types.String `tfsdk:"repo_type"`
	IsPublic                  types.Bool   `tfsdk:"is_public"`
	AllowAnonymousAccess      types.Bool   `tfsdk:"allow_anonymous_access"`
	StorageUsedBytes          types.Int64  `tfsdk:"storage_used_bytes"`
	QuotaBytes                types.Int64  `tfsdk:"quota_bytes"`
	UpstreamURL               types.String `tfsdk:"upstream_url"`
	UpstreamAuthType          types.String `tfsdk:"upstream_auth_type"`
	UpstreamAuthConfigured    types.Bool   `tfsdk:"upstream_auth_configured"`
	CreatedAt                 types.String `tfsdk:"created_at"`
	UpdatedAt                 types.String `tfsdk:"updated_at"`
	Members                   types.List   `tfsdk:"members"`
	PromotionOnly             types.Bool   `tfsdk:"promotion_only"`
	VersioningEnabled         types.Bool   `tfsdk:"versioning_enabled"`
	ProjectID                 types.String `tfsdk:"project_id"`
	HasTrustedGpgKey          types.Bool   `tfsdk:"has_trusted_gpg_key"`
	TrustedGpgKey             types.String `tfsdk:"trusted_gpg_key"`
	CustomUserAgent           types.String `tfsdk:"custom_user_agent"`
	StorageBackend            types.String `tfsdk:"storage_backend"`
	FormatKey                 types.String `tfsdk:"format_key"`
	IndexUpstreamURL          types.String `tfsdk:"index_upstream_url"`
	PypiUpstreamIndexPath     types.String `tfsdk:"pypi_upstream_index_path"`
	AptOrigin                 types.String `tfsdk:"apt_origin"`
	AptLabel                  types.String `tfsdk:"apt_label"`
	AptReleaseVersion         types.String `tfsdk:"apt_release_version"`
	AptDescription            types.String `tfsdk:"apt_description"`
	QuarantineEnabled         types.Bool   `tfsdk:"quarantine_enabled"`
	QuarantineDurationMinutes types.Int64  `tfsdk:"quarantine_duration_minutes"`
	CurationEnabled           types.Bool   `tfsdk:"curation_enabled"`
	CurationDefaultAction     types.String `tfsdk:"curation_default_action"`
	CurationAllowUnverified   types.Bool   `tfsdk:"curation_allow_unverified"`
	NpmAllowedNamePatterns    types.List   `tfsdk:"npm_allowed_name_patterns"`
	Debian                    types.Object `tfsdk:"debian"`
}

// debianConfigModel maps the nested `debian` block (the Debian remote proxy
// filter). Mirrors client.DebianConfig field-for-field.
type debianConfigModel struct {
	DistributionPaths      types.List   `tfsdk:"distribution_paths"`
	Components             types.List   `tfsdk:"components"`
	Architectures          types.List   `tfsdk:"architectures"`
	IncludeSourcePackages  types.Bool   `tfsdk:"include_source_packages"`
	FlatRepository         types.Bool   `tfsdk:"flat_repository"`
	VerifyUpstreamMetadata types.Bool   `tfsdk:"verify_upstream_metadata"`
	UpstreamGpgKeyID       types.String `tfsdk:"upstream_gpg_key_id"`
	MetadataStrategy       types.String `tfsdk:"metadata_strategy"`
	PackageFetchStrategy   types.String `tfsdk:"package_fetch_strategy"`
	PackageQueries         types.List   `tfsdk:"package_queries"`
	AllowEncodedSeparators types.Bool   `tfsdk:"allow_encoded_separators"`
}

var debianAttrTypes = map[string]attr.Type{
	"distribution_paths":       types.ListType{ElemType: types.StringType},
	"components":               types.ListType{ElemType: types.StringType},
	"architectures":            types.ListType{ElemType: types.StringType},
	"include_source_packages":  types.BoolType,
	"flat_repository":          types.BoolType,
	"verify_upstream_metadata": types.BoolType,
	"upstream_gpg_key_id":      types.StringType,
	"metadata_strategy":        types.StringType,
	"package_fetch_strategy":   types.StringType,
	"package_queries":          types.ListType{ElemType: types.StringType},
	"allow_encoded_separators": types.BoolType,
}

func (r *repositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *repositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a repository in Artifact Keeper. Destroying this resource deletes the repository and every artifact stored in it, guard production repositories with `lifecycle { prevent_destroy = true }`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Repository UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Stable repository key used in API paths (e.g. `docker-local`). Changing this forces a new repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable display name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Free-form description of the repository.",
			},
			"format": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Package format, e.g. `docker`, `npm`, `maven`, `pypi`, `generic`. Changing this forces a new repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repo_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Repository type: `local`, `remote`, `virtual`, or `staging`. Changing this forces a new repository.",
				Validators:          []validator.String{stringvalidator.OneOf("local", "remote", "virtual", "staging")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"is_public": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether anonymous (unauthenticated) users may download artifacts. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"allow_anonymous_access": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Convenience alias reported by the API; always equal to `is_public`.",
			},
			"storage_used_bytes": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Current storage consumed by the repository, in bytes.",
			},
			"quota_bytes": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Storage quota in bytes. Omit for unlimited.",
			},
			"upstream_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Upstream registry URL for `remote` (pull-through cache) repositories. Changing this forces a new repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"upstream_auth_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Configured upstream auth type (`basic` or `bearer`), if any. Managed via the upstream-auth API, not this resource.",
			},
			"upstream_auth_configured": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether upstream credentials are configured for this repository.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 last-update timestamp.",
			},
			"members": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "For `virtual` repositories: ordered list of member repository keys to aggregate. List order sets resolution priority (first = highest). Managed authoritatively when set; omit to leave membership unmanaged. Not valid for non-virtual repositories.",
			},
			"promotion_only": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When true, direct uploads are rejected, artifacts must arrive via promotion. Admin-only. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"versioning_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Enable first-class artifact versioning (Generic/Mlmodel repos append immutable revisions instead of overwriting). Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UUID of a project to assign this repository to; project-level grants are inherited. Omit to leave unassigned.",
			},
			"custom_user_agent": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom `User-Agent` for outbound requests to the upstream (`remote` repos only). Max 256 characters.",
			},
			"apt_origin": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom `Origin` field for Debian/APT `Release` files.",
			},
			"apt_label": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom `Label` field for Debian/APT `Release` files.",
			},
			"apt_release_version": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom `Version` field for Debian/APT `Release` files.",
			},
			"apt_description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom `Description` field for Debian/APT `Release` files.",
			},
			"trusted_gpg_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "ASCII-armored OpenPGP **public** key trusted to sign an RPM curation remote's `repomd.xml`. Write-only, the API never returns it (see `has_trusted_gpg_key`).",
			},
			"has_trusted_gpg_key": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether a trusted upstream GPG public key is configured.",
			},
			"index_upstream_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Separate index host for Cargo registries that split index and downloads (e.g. `https://index.crates.io`). Write-only in the API.",
			},
			"pypi_upstream_index_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "PyPI simple-index prefix override for non-PEP 503 upstreams: `\"simple\"` (default), `\"\"` for a flat CDN, or a custom prefix. Write-only in the API.",
			},
			"storage_backend": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Override the storage backend for this repository. Non-admins may only use the default. Changing this forces a new repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"format_key": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom format key for a WASM plugin format handler (e.g. `rpm-custom`). Changing this forces a new repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"quarantine_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Hold newly uploaded artifacts in quarantine until they are scanned. Defaults to the instance-wide setting when unset.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"quarantine_duration_minutes": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Quarantine hold duration in minutes. Only meaningful when `quarantine_enabled` is true.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"curation_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Enforce curation rules on this repository's proxy paths (blocks downloads that fail curation). Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"curation_default_action": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Stance applied when no curation rule matches: `allow` or `review`. (`block` is rejected; use block rules for specific packages.)",
				Validators:          []validator.String{stringvalidator.OneOf("allow", "review")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"curation_allow_unverified": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Opt into ingesting unverified upstream metadata on the keyless RPM curation-sync path. Omit or `false` for the fail-closed default. Write-only, the API never returns it.",
			},
			"npm_allowed_name_patterns": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Allowed npm full-name glob patterns (`*`/`?`) for an npm `remote` repository, e.g. `@acme/*` or `internal-*`. Additive to the scope allow-list managed by `artifactkeeper_repository_npm_scope_policy`.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"debian": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Debian remote (proxy) distribution/component/architecture filter. Only valid for Debian `remote` repositories. Empty or `[\"*\"]` lists proxy everything. Unset fields keep the server default; once the block is set, widen it by setting empty lists rather than dropping the block.",
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"distribution_paths": schema.ListAttribute{
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Distributions (suites/codenames, e.g. `bookworm`) to proxy. Empty or `[\"*\"]` = all.",
						PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
					},
					"components": schema.ListAttribute{
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Components (e.g. `main`, `contrib`, `non-free`) to proxy. Empty or `[\"*\"]` = all.",
						PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
					},
					"architectures": schema.ListAttribute{
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Architectures (e.g. `amd64`, `arm64`) to proxy. Empty or `[\"*\"]` = all; `all` is always permitted.",
						PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
					},
					"include_source_packages": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Whether source packages (`Sources`, `.dsc`) are proxied.",
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
					"flat_repository": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Flat (non-`dists/`) repository layout flag.",
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
					"verify_upstream_metadata": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Verify the upstream signed `Release` before serving (reserved for later trust-anchor verification).",
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
					"upstream_gpg_key_id": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Upstream GPG key id used to verify upstream metadata.",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"metadata_strategy": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Metadata production strategy (e.g. `upstream_passthrough`).",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"package_fetch_strategy": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Package-body fetch strategy (e.g. `upstream_passthrough`).",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"package_queries": schema.ListAttribute{
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Package-name-level queries. Not honored in passthrough mode (a non-empty value with `upstream_passthrough` is rejected).",
						PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
					},
					"allow_encoded_separators": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Allow percent-encoded path separators in proxy request paths. Defaults to `false` (path-confusion probes are rejected).",
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
				},
			},
		},
	}
}

func (r *repositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *repositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateRepositoryRequest{
		Key:      plan.Key.ValueString(),
		Name:     plan.Name.ValueString(),
		Format:   plan.Format.ValueString(),
		RepoType: plan.RepoType.ValueString(),
	}
	if !plan.Description.IsNull() {
		createReq.Description = plan.Description.ValueStringPointer()
	}
	if !plan.IsPublic.IsNull() && !plan.IsPublic.IsUnknown() {
		createReq.IsPublic = plan.IsPublic.ValueBoolPointer()
	}
	if !plan.UpstreamURL.IsNull() {
		createReq.UpstreamURL = plan.UpstreamURL.ValueStringPointer()
	}
	if !plan.QuotaBytes.IsNull() {
		createReq.QuotaBytes = plan.QuotaBytes.ValueInt64Pointer()
	}
	if !plan.PromotionOnly.IsNull() && !plan.PromotionOnly.IsUnknown() {
		createReq.PromotionOnly = plan.PromotionOnly.ValueBoolPointer()
	}
	if !plan.VersioningEnabled.IsNull() && !plan.VersioningEnabled.IsUnknown() {
		createReq.VersioningEnabled = plan.VersioningEnabled.ValueBoolPointer()
	}
	if !plan.ProjectID.IsNull() {
		createReq.ProjectID = plan.ProjectID.ValueStringPointer()
	}
	if !plan.CustomUserAgent.IsNull() {
		createReq.CustomUserAgent = plan.CustomUserAgent.ValueStringPointer()
	}
	if !plan.TrustedGpgKey.IsNull() {
		createReq.TrustedGpgKey = plan.TrustedGpgKey.ValueStringPointer()
	}
	if !plan.StorageBackend.IsNull() {
		createReq.StorageBackend = plan.StorageBackend.ValueStringPointer()
	}
	if !plan.FormatKey.IsNull() {
		createReq.FormatKey = plan.FormatKey.ValueStringPointer()
	}
	if !plan.IndexUpstreamURL.IsNull() {
		createReq.IndexUpstreamURL = plan.IndexUpstreamURL.ValueStringPointer()
	}
	if !plan.PypiUpstreamIndexPath.IsNull() {
		createReq.PypiUpstreamIndexPath = plan.PypiUpstreamIndexPath.ValueStringPointer()
	}
	if !plan.AptOrigin.IsNull() {
		createReq.AptOrigin = plan.AptOrigin.ValueStringPointer()
	}
	if !plan.AptLabel.IsNull() {
		createReq.AptLabel = plan.AptLabel.ValueStringPointer()
	}
	if !plan.AptReleaseVersion.IsNull() {
		createReq.AptReleaseVersion = plan.AptReleaseVersion.ValueStringPointer()
	}
	if !plan.AptDescription.IsNull() {
		createReq.AptDescription = plan.AptDescription.ValueStringPointer()
	}
	if !plan.CurationAllowUnverified.IsNull() && !plan.CurationAllowUnverified.IsUnknown() {
		createReq.CurationAllowUnverified = plan.CurationAllowUnverified.ValueBoolPointer()
	}
	if !plan.NpmAllowedNamePatterns.IsNull() && !plan.NpmAllowedNamePatterns.IsUnknown() {
		patterns, d := listToStringSlice(ctx, plan.NpmAllowedNamePatterns)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.NpmAllowedNamePatterns = patterns
	}
	debianReq, diags := debianConfigFromObject(ctx, plan.Debian)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createReq.Debian = debianReq

	if !plan.Members.IsNull() && plan.RepoType.ValueString() != "virtual" {
		resp.Diagnostics.AddAttributeError(
			path.Root("members"),
			"members not allowed",
			"`members` is only valid for repositories with repo_type = \"virtual\".",
		)
		return
	}

	repo, err := r.client.CreateRepository(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating repository", err.Error())
		return
	}

	// quarantine and curation enforcement are update-only server-side, so apply
	// them with a follow-up PATCH when the user set any of them.
	if postReq, ok := repoPostCreateUpdate(plan); ok {
		repo, err = r.client.UpdateRepository(ctx, plan.Key.ValueString(), postReq)
		if err != nil {
			resp.Diagnostics.AddError("Error applying repository settings", err.Error())
			return
		}
	}

	model, diags := repositoryToModel(ctx, repo)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	applyWriteOnlyRepoFields(&model, plan)
	if !plan.Members.IsNull() {
		var keys []string
		resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &keys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SetVirtualMembers(ctx, plan.Key.ValueString(), keys); err != nil {
			resp.Diagnostics.AddError("Error setting virtual repository members", err.Error())
			return
		}
		model.Members = plan.Members
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *repositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repo, err := r.client.GetRepository(ctx, state.Key.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository", err.Error())
		return
	}

	model, diags := repositoryToModel(ctx, repo)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	applyWriteOnlyRepoFields(&model, state)
	// Only reconcile membership when configured; leaving it null avoids a
	// null-vs-empty diff for repos that don't use it.
	if repo.RepoType == "virtual" && !state.Members.IsNull() {
		members, err := r.client.GetVirtualMembers(ctx, state.Key.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading virtual repository members", err.Error())
			return
		}
		keys := make([]string, len(members))
		for i, m := range members {
			keys[i] = m.MemberRepoKey
		}
		lv, d := types.ListValueFrom(ctx, types.StringType, keys)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		model.Members = lv
	} else {
		model.Members = state.Members
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *repositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateRepositoryRequest{
		Name: plan.Name.ValueStringPointer(),
	}
	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueStringPointer()
	}
	if !plan.IsPublic.IsNull() && !plan.IsPublic.IsUnknown() {
		updateReq.IsPublic = plan.IsPublic.ValueBoolPointer()
	}
	if !plan.QuotaBytes.IsNull() {
		updateReq.QuotaBytes = plan.QuotaBytes.ValueInt64Pointer()
	}
	if !plan.PromotionOnly.IsNull() && !plan.PromotionOnly.IsUnknown() {
		updateReq.PromotionOnly = plan.PromotionOnly.ValueBoolPointer()
	}
	if !plan.VersioningEnabled.IsNull() && !plan.VersioningEnabled.IsUnknown() {
		updateReq.VersioningEnabled = plan.VersioningEnabled.ValueBoolPointer()
	}
	if !plan.ProjectID.IsNull() {
		updateReq.ProjectID = plan.ProjectID.ValueStringPointer()
	}
	if !plan.CustomUserAgent.IsNull() {
		updateReq.CustomUserAgent = plan.CustomUserAgent.ValueStringPointer()
	}
	if !plan.TrustedGpgKey.IsNull() {
		updateReq.TrustedGpgKey = plan.TrustedGpgKey.ValueStringPointer()
	}
	if !plan.IndexUpstreamURL.IsNull() {
		updateReq.IndexUpstreamURL = plan.IndexUpstreamURL.ValueStringPointer()
	}
	if !plan.PypiUpstreamIndexPath.IsNull() {
		updateReq.PypiUpstreamIndexPath = plan.PypiUpstreamIndexPath.ValueStringPointer()
	}
	if !plan.AptOrigin.IsNull() {
		updateReq.AptOrigin = plan.AptOrigin.ValueStringPointer()
	}
	if !plan.AptLabel.IsNull() {
		updateReq.AptLabel = plan.AptLabel.ValueStringPointer()
	}
	if !plan.AptReleaseVersion.IsNull() {
		updateReq.AptReleaseVersion = plan.AptReleaseVersion.ValueStringPointer()
	}
	if !plan.AptDescription.IsNull() {
		updateReq.AptDescription = plan.AptDescription.ValueStringPointer()
	}
	if !plan.QuarantineEnabled.IsNull() && !plan.QuarantineEnabled.IsUnknown() {
		updateReq.QuarantineEnabled = plan.QuarantineEnabled.ValueBoolPointer()
	}
	if !plan.QuarantineDurationMinutes.IsNull() && !plan.QuarantineDurationMinutes.IsUnknown() {
		updateReq.QuarantineDurationMinutes = plan.QuarantineDurationMinutes.ValueInt64Pointer()
	}
	if !plan.CurationEnabled.IsNull() && !plan.CurationEnabled.IsUnknown() {
		updateReq.CurationEnabled = plan.CurationEnabled.ValueBoolPointer()
	}
	if !plan.CurationDefaultAction.IsNull() && !plan.CurationDefaultAction.IsUnknown() {
		updateReq.CurationDefaultAction = plan.CurationDefaultAction.ValueStringPointer()
	}
	if !plan.CurationAllowUnverified.IsNull() && !plan.CurationAllowUnverified.IsUnknown() {
		updateReq.CurationAllowUnverified = plan.CurationAllowUnverified.ValueBoolPointer()
	}
	if !plan.NpmAllowedNamePatterns.IsNull() && !plan.NpmAllowedNamePatterns.IsUnknown() {
		patterns, d := listToStringSlice(ctx, plan.NpmAllowedNamePatterns)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.NpmAllowedNamePatterns = patterns
	}
	debianReq, diags := debianConfigFromObject(ctx, plan.Debian)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq.Debian = debianReq

	if !plan.Members.IsNull() && plan.RepoType.ValueString() != "virtual" {
		resp.Diagnostics.AddAttributeError(
			path.Root("members"),
			"members not allowed",
			"`members` is only valid for repositories with repo_type = \"virtual\".",
		)
		return
	}

	repo, err := r.client.UpdateRepository(ctx, plan.Key.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating repository", err.Error())
		return
	}

	model, diags := repositoryToModel(ctx, repo)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	applyWriteOnlyRepoFields(&model, plan)
	if !plan.Members.IsNull() {
		var keys []string
		resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &keys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SetVirtualMembers(ctx, plan.Key.ValueString(), keys); err != nil {
			resp.Diagnostics.AddError("Error setting virtual repository members", err.Error())
			return
		}
		model.Members = plan.Members
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *repositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRepository(ctx, state.Key.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting repository", err.Error())
	}
}

func (r *repositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Repositories are addressed by key in the API; import by key.
	resource.ImportStatePassthroughID(ctx, path.Root("key"), req, resp)
}

func repositoryToModel(ctx context.Context, repo *client.Repository) (repositoryResourceModel, diag.Diagnostics) {
	debian, diags := debianConfigToObject(ctx, repo.Debian)
	npmPatterns, d := stringListValue(ctx, repo.NpmAllowedNamePatterns)
	diags.Append(d...)
	model := repositoryResourceModel{
		ID:                        types.StringValue(repo.ID),
		Key:                       types.StringValue(repo.Key),
		Name:                      types.StringValue(repo.Name),
		Description:               stringPointerValue(repo.Description),
		Format:                    types.StringValue(repo.Format),
		RepoType:                  types.StringValue(repo.RepoType),
		IsPublic:                  types.BoolValue(repo.IsPublic),
		AllowAnonymousAccess:      types.BoolValue(repo.AllowAnonymousAccess),
		StorageUsedBytes:          types.Int64Value(repo.StorageUsedBytes),
		QuotaBytes:                int64PointerValue(repo.QuotaBytes),
		UpstreamURL:               stringPointerValue(repo.UpstreamURL),
		UpstreamAuthType:          stringPointerValue(repo.UpstreamAuthType),
		UpstreamAuthConfigured:    types.BoolValue(repo.UpstreamAuthConfigured),
		CreatedAt:                 types.StringValue(repo.CreatedAt),
		UpdatedAt:                 types.StringValue(repo.UpdatedAt),
		Members:                   types.ListNull(types.StringType),
		PromotionOnly:             types.BoolValue(repo.PromotionOnly),
		VersioningEnabled:         types.BoolValue(repo.VersioningEnabled),
		ProjectID:                 stringPointerValue(repo.ProjectID),
		HasTrustedGpgKey:          types.BoolValue(repo.HasTrustedGpgKey),
		CustomUserAgent:           stringPointerValue(repo.CustomUserAgent),
		AptOrigin:                 stringPointerValue(repo.AptOrigin),
		AptLabel:                  stringPointerValue(repo.AptLabel),
		AptReleaseVersion:         stringPointerValue(repo.AptReleaseVersion),
		AptDescription:            stringPointerValue(repo.AptDescription),
		QuarantineEnabled:         boolPointerValue(repo.QuarantineEnabled),
		QuarantineDurationMinutes: int64PointerValue(repo.QuarantineDurationMinutes),
		CurationEnabled:           types.BoolValue(repo.CurationEnabled),
		CurationDefaultAction:     types.StringValue(repo.CurationDefaultAction),
		NpmAllowedNamePatterns:    npmPatterns,
		Debian:                    debian,
		// Write-only/create-only fields the API never returns; filled from plan
		// (Create/Update) or prior state (Read).
		TrustedGpgKey:           types.StringNull(),
		StorageBackend:          types.StringNull(),
		FormatKey:               types.StringNull(),
		IndexUpstreamURL:        types.StringNull(),
		PypiUpstreamIndexPath:   types.StringNull(),
		CurationAllowUnverified: types.BoolNull(),
	}
	return model, diags
}

// repoPostCreateUpdate builds a PATCH for the update-only settable fields
// (quarantine, curation enforcement) the create endpoint doesn't accept,
// returning false when the user set none of them.
func repoPostCreateUpdate(plan repositoryResourceModel) (client.UpdateRepositoryRequest, bool) {
	var req client.UpdateRepositoryRequest
	set := false
	if !plan.QuarantineEnabled.IsNull() && !plan.QuarantineEnabled.IsUnknown() {
		req.QuarantineEnabled = plan.QuarantineEnabled.ValueBoolPointer()
		set = true
	}
	if !plan.QuarantineDurationMinutes.IsNull() && !plan.QuarantineDurationMinutes.IsUnknown() {
		req.QuarantineDurationMinutes = plan.QuarantineDurationMinutes.ValueInt64Pointer()
		set = true
	}
	if !plan.CurationEnabled.IsNull() && !plan.CurationEnabled.IsUnknown() {
		req.CurationEnabled = plan.CurationEnabled.ValueBoolPointer()
		set = true
	}
	if !plan.CurationDefaultAction.IsNull() && !plan.CurationDefaultAction.IsUnknown() {
		req.CurationDefaultAction = plan.CurationDefaultAction.ValueStringPointer()
		set = true
	}
	return req, set
}

// debianConfigToObject converts the client Debian filter into the nested object
// value (null when the repository has no filter).
func debianConfigToObject(ctx context.Context, d *client.DebianConfig) (types.Object, diag.Diagnostics) {
	if d == nil {
		return types.ObjectNull(debianAttrTypes), nil
	}
	var diags diag.Diagnostics
	dp, d1 := stringListValue(ctx, d.DistributionPaths)
	comp, d2 := stringListValue(ctx, d.Components)
	arch, d3 := stringListValue(ctx, d.Architectures)
	pq, d4 := stringListValue(ctx, d.PackageQueries)
	diags.Append(d1...)
	diags.Append(d2...)
	diags.Append(d3...)
	diags.Append(d4...)
	m := debianConfigModel{
		DistributionPaths:      dp,
		Components:             comp,
		Architectures:          arch,
		IncludeSourcePackages:  types.BoolValue(d.IncludeSourcePackages),
		FlatRepository:         types.BoolValue(d.FlatRepository),
		VerifyUpstreamMetadata: types.BoolValue(d.VerifyUpstreamMetadata),
		UpstreamGpgKeyID:       stringPointerValue(d.UpstreamGpgKeyID),
		MetadataStrategy:       types.StringValue(d.MetadataStrategy),
		PackageFetchStrategy:   types.StringValue(d.PackageFetchStrategy),
		PackageQueries:         pq,
		AllowEncodedSeparators: types.BoolValue(d.AllowEncodedSeparators),
	}
	obj, d5 := types.ObjectValueFrom(ctx, debianAttrTypes, m)
	diags.Append(d5...)
	return obj, diags
}

// debianConfigFromObject converts the nested `debian` block into a client
// request payload (nil when the block is absent). Unknown/unset fields are left
// off so the server default applies.
func debianConfigFromObject(ctx context.Context, o types.Object) (*client.DebianConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	if o.IsNull() || o.IsUnknown() {
		return nil, diags
	}
	var m debianConfigModel
	diags.Append(o.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	dp, d1 := listToStringSlice(ctx, m.DistributionPaths)
	comp, d2 := listToStringSlice(ctx, m.Components)
	arch, d3 := listToStringSlice(ctx, m.Architectures)
	pq, d4 := listToStringSlice(ctx, m.PackageQueries)
	diags.Append(d1...)
	diags.Append(d2...)
	diags.Append(d3...)
	diags.Append(d4...)
	cfg := &client.DebianConfig{
		DistributionPaths:      dp,
		Components:             comp,
		Architectures:          arch,
		IncludeSourcePackages:  m.IncludeSourcePackages.ValueBool(),
		FlatRepository:         m.FlatRepository.ValueBool(),
		VerifyUpstreamMetadata: m.VerifyUpstreamMetadata.ValueBool(),
		MetadataStrategy:       m.MetadataStrategy.ValueString(),
		PackageFetchStrategy:   m.PackageFetchStrategy.ValueString(),
		PackageQueries:         pq,
		AllowEncodedSeparators: m.AllowEncodedSeparators.ValueBool(),
	}
	if !m.UpstreamGpgKeyID.IsNull() && !m.UpstreamGpgKeyID.IsUnknown() {
		cfg.UpstreamGpgKeyID = m.UpstreamGpgKeyID.ValueStringPointer()
	}
	return cfg, diags
}

// applyWriteOnlyRepoFields copies write-only/create-only fields (never returned
// by the API) from src to dst, so Read keeps prior state and Create/Update the plan.
func applyWriteOnlyRepoFields(dst *repositoryResourceModel, src repositoryResourceModel) {
	dst.TrustedGpgKey = src.TrustedGpgKey
	dst.StorageBackend = src.StorageBackend
	dst.FormatKey = src.FormatKey
	dst.IndexUpstreamURL = src.IndexUpstreamURL
	dst.PypiUpstreamIndexPath = src.PypiUpstreamIndexPath
	dst.CurationAllowUnverified = src.CurationAllowUnverified
}

func stringPointerValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func int64PointerValue(i *int64) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*i)
}

func boolPointerValue(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}
