package client

import (
	"context"
	"net/http"
	"net/url"
)

// Repository mirrors RepositoryResponse (GET /repositories/{key}).
type Repository struct {
	ID                     string  `json:"id"`
	Key                    string  `json:"key"`
	Name                   string  `json:"name"`
	Description            *string `json:"description"`
	Format                 string  `json:"format"`
	RepoType               string  `json:"repo_type"`
	IsPublic               bool    `json:"is_public"`
	AllowAnonymousAccess   bool    `json:"allow_anonymous_access"`
	StorageUsedBytes       int64   `json:"storage_used_bytes"`
	QuotaBytes             *int64  `json:"quota_bytes"`
	UpstreamURL            *string `json:"upstream_url"`
	UpstreamAuthType       *string `json:"upstream_auth_type"`
	UpstreamAuthConfigured bool    `json:"upstream_auth_configured"`
	PromotionOnly          bool    `json:"promotion_only"`
	VersioningEnabled      bool    `json:"versioning_enabled"`
	ProjectID              *string `json:"project_id"`
	HasTrustedGpgKey       bool    `json:"has_trusted_gpg_key"`
	CustomUserAgent        *string `json:"custom_user_agent"`
	AptOrigin              *string `json:"apt_origin"`
	AptLabel               *string `json:"apt_label"`
	AptReleaseVersion      *string `json:"apt_release_version"`
	AptDescription         *string `json:"apt_description"`
	// Quarantine hold (#1770): nil when unset (global default applies).
	QuarantineEnabled         *bool  `json:"quarantine_enabled"`
	QuarantineDurationMinutes *int64 `json:"quarantine_duration_minutes"`
	// Curation enforcement on this repo's proxy paths (#2912). Stored on the
	// repositories row, so always present.
	CurationEnabled       bool   `json:"curation_enabled"`
	CurationDefaultAction string `json:"curation_default_action"`
	// Allowed npm full-name glob patterns for an npm remote (#2424). The scope
	// allow-list itself is managed by the repository_npm_scope_policy resource;
	// only the name-pattern list is reachable via the repository object.
	NpmAllowedNamePatterns []string      `json:"npm_allowed_name_patterns"`
	Debian                 *DebianConfig `json:"debian"`
	CreatedAt              string        `json:"created_at"`
	UpdatedAt              string        `json:"updated_at"`
}

// DebianConfig is the Debian remote (proxy) distribution/component/architecture
// filter (#2460), stored under `debian_config`. Empty lists (or `["*"]`) proxy
// everything. Only meaningful for Debian remote repositories. On update the
// whole object is sent as a patch, which the backend merges over the stored
// config, so sending every field makes it authoritative (a full replace).
type DebianConfig struct {
	DistributionPaths      []string `json:"distribution_paths"`
	Components             []string `json:"components"`
	Architectures          []string `json:"architectures"`
	IncludeSourcePackages  bool     `json:"include_source_packages"`
	FlatRepository         bool     `json:"flat_repository"`
	VerifyUpstreamMetadata bool     `json:"verify_upstream_metadata"`
	UpstreamGpgKeyID       *string  `json:"upstream_gpg_key_id,omitempty"`
	// Enum strings; omitempty so an unset value falls back to the server default
	// (`upstream_passthrough`) rather than failing to parse an empty string.
	MetadataStrategy       string   `json:"metadata_strategy,omitempty"`
	PackageFetchStrategy   string   `json:"package_fetch_strategy,omitempty"`
	PackageQueries         []string `json:"package_queries"`
	AllowEncodedSeparators bool     `json:"allow_encoded_separators"`
}

// CreateRepositoryRequest maps CreateRepositoryRequest. Only the fields the
// Terraform resource surfaces; the API accepts more.
type CreateRepositoryRequest struct {
	Key                   string  `json:"key"`
	Name                  string  `json:"name"`
	Description           *string `json:"description,omitempty"`
	Format                string  `json:"format"`
	RepoType              string  `json:"repo_type"`
	IsPublic              *bool   `json:"is_public,omitempty"`
	UpstreamURL           *string `json:"upstream_url,omitempty"`
	QuotaBytes            *int64  `json:"quota_bytes,omitempty"`
	PromotionOnly         *bool   `json:"promotion_only,omitempty"`
	VersioningEnabled     *bool   `json:"versioning_enabled,omitempty"`
	StorageBackend        *string `json:"storage_backend,omitempty"`
	FormatKey             *string `json:"format_key,omitempty"`
	IndexUpstreamURL      *string `json:"index_upstream_url,omitempty"`
	PypiUpstreamIndexPath *string `json:"pypi_upstream_index_path,omitempty"`
	CustomUserAgent       *string `json:"custom_user_agent,omitempty"`
	ProjectID             *string `json:"project_id,omitempty"`
	TrustedGpgKey         *string `json:"trusted_gpg_key,omitempty"`
	// Opt into ingesting unverified upstream metadata on the keyless RPM
	// curation-sync path (#2569). Write-only; not echoed in the response.
	CurationAllowUnverified *bool         `json:"curation_allow_unverified,omitempty"`
	NpmAllowedNamePatterns  []string      `json:"npm_allowed_name_patterns,omitempty"`
	Debian                  *DebianConfig `json:"debian,omitempty"`
	AptOrigin               *string       `json:"apt_origin,omitempty"`
	AptLabel                *string       `json:"apt_label,omitempty"`
	AptReleaseVersion       *string       `json:"apt_release_version,omitempty"`
	AptDescription          *string       `json:"apt_description,omitempty"`
}

// UpdateRepositoryRequest maps the mutable subset of UpdateRepositoryRequest
// (PATCH /repositories/{key}).
type UpdateRepositoryRequest struct {
	Name                  *string `json:"name,omitempty"`
	Description           *string `json:"description,omitempty"`
	IsPublic              *bool   `json:"is_public,omitempty"`
	QuotaBytes            *int64  `json:"quota_bytes,omitempty"`
	PromotionOnly         *bool   `json:"promotion_only,omitempty"`
	VersioningEnabled     *bool   `json:"versioning_enabled,omitempty"`
	IndexUpstreamURL      *string `json:"index_upstream_url,omitempty"`
	PypiUpstreamIndexPath *string `json:"pypi_upstream_index_path,omitempty"`
	CustomUserAgent       *string `json:"custom_user_agent,omitempty"`
	ProjectID             *string `json:"project_id,omitempty"`
	TrustedGpgKey         *string `json:"trusted_gpg_key,omitempty"`
	// Update-only settable fields (not accepted on create; set with a follow-up
	// PATCH). All echoed back in the response, so they round-trip.
	QuarantineEnabled         *bool    `json:"quarantine_enabled,omitempty"`
	QuarantineDurationMinutes *int64   `json:"quarantine_duration_minutes,omitempty"`
	CurationAllowUnverified   *bool    `json:"curation_allow_unverified,omitempty"`
	CurationEnabled           *bool    `json:"curation_enabled,omitempty"`
	CurationDefaultAction     *string  `json:"curation_default_action,omitempty"`
	NpmAllowedNamePatterns    []string `json:"npm_allowed_name_patterns,omitempty"`
	// Whole object sent as a patch; the backend merges it over the stored
	// config, so sending every field makes it authoritative.
	Debian            *DebianConfig `json:"debian,omitempty"`
	AptOrigin         *string       `json:"apt_origin,omitempty"`
	AptLabel          *string       `json:"apt_label,omitempty"`
	AptReleaseVersion *string       `json:"apt_release_version,omitempty"`
	AptDescription    *string       `json:"apt_description,omitempty"`
}

func (c *Client) CreateRepository(ctx context.Context, req CreateRepositoryRequest) (*Repository, error) {
	var out Repository
	if err := c.do(ctx, http.MethodPost, "/repositories", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetRepository(ctx context.Context, key string) (*Repository, error) {
	var out Repository
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(key), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateRepository(ctx context.Context, key string, req UpdateRepositoryRequest) (*Repository, error) {
	var out Repository
	if err := c.do(ctx, http.MethodPatch, "/repositories/"+url.PathEscape(key), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRepository(ctx context.Context, key string) error {
	return c.do(ctx, http.MethodDelete, "/repositories/"+url.PathEscape(key), nil, nil)
}

// VirtualMember is an entry of GET /repositories/{key}/members, ordered by
// priority ascending.
type VirtualMember struct {
	MemberRepoKey string `json:"member_repo_key"`
	Priority      int    `json:"priority"`
}

type virtualMembersListResponse struct {
	Members []VirtualMember `json:"members"`
}

// virtualMemberPriority is one entry of the PUT /members body.
type virtualMemberPriority struct {
	MemberKey string `json:"member_key"`
	Priority  int    `json:"priority"`
}

type updateVirtualMembersRequest struct {
	Members []virtualMemberPriority `json:"members"`
}

// GetVirtualMembers lists a virtual repository's member keys, ordered by
// priority.
func (c *Client) GetVirtualMembers(ctx context.Context, key string) ([]VirtualMember, error) {
	var out virtualMembersListResponse
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(key)+"/members", nil, &out); err != nil {
		return nil, err
	}
	return out.Members, nil
}

// SetVirtualMembers replaces the full member set (PUT /members). Priority
// follows list position (1-based), so the first key resolves first.
func (c *Client) SetVirtualMembers(ctx context.Context, key string, memberKeys []string) error {
	body := updateVirtualMembersRequest{Members: make([]virtualMemberPriority, len(memberKeys))}
	for i, mk := range memberKeys {
		body.Members[i] = virtualMemberPriority{MemberKey: mk, Priority: i + 1}
	}
	return c.do(ctx, http.MethodPut, "/repositories/"+url.PathEscape(key)+"/members", body, nil)
}
