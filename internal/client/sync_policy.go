package client

import (
	"context"
	"net/http"
	"net/url"
)

// RepoSelector matches repositories for a sync policy (RepoSelectorSchema).
// All non-empty fields combine with AND semantics. Used for both request
// encoding (empty fields are omitted) and response decoding.
type RepoSelector struct {
	MatchLabels  map[string]string `json:"match_labels,omitempty"`
	MatchFormats []string          `json:"match_formats,omitempty"`
	MatchPattern *string           `json:"match_pattern,omitempty"`
	MatchRepos   []string          `json:"match_repos,omitempty"`
}

// PeerSelector matches peer instances for a sync policy (PeerSelectorSchema).
type PeerSelector struct {
	All         bool              `json:"all,omitempty"`
	MatchLabels map[string]string `json:"match_labels,omitempty"`
	MatchRegion *string           `json:"match_region,omitempty"`
	MatchPeers  []string          `json:"match_peers,omitempty"`
}

// ArtifactFilter constrains which artifacts a sync policy replicates
// (ArtifactFilterSchema).
type ArtifactFilter struct {
	MaxAgeDays   *int64            `json:"max_age_days,omitempty"`
	IncludePaths []string          `json:"include_paths,omitempty"`
	ExcludePaths []string          `json:"exclude_paths,omitempty"`
	MaxSizeBytes *int64            `json:"max_size_bytes,omitempty"`
	MatchTags    map[string]string `json:"match_tags,omitempty"`
}

// SyncPolicy mirrors SyncPolicyResponse (GET /sync-policies/{id}). The API
// always serializes the selectors as fully-populated objects (empty fields
// present), so the typed selectors above decode cleanly.
type SyncPolicy struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Enabled         bool           `json:"enabled"`
	RepoSelector    RepoSelector   `json:"repo_selector"`
	PeerSelector    PeerSelector   `json:"peer_selector"`
	ReplicationMode string         `json:"replication_mode"`
	Priority        int64          `json:"priority"`
	ArtifactFilter  ArtifactFilter `json:"artifact_filter"`
	Filter          string         `json:"filter"`
	Precedence      int64          `json:"precedence"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
}

// SyncPolicyRequest maps the create (POST) and update (PUT) payloads. Scalar
// fields with server-side defaults use pointers so an omitted value falls back
// to the API default; selectors are always sent so an update can clear them.
type SyncPolicyRequest struct {
	Name            string          `json:"name"`
	Description     *string         `json:"description,omitempty"`
	Enabled         *bool           `json:"enabled,omitempty"`
	RepoSelector    *RepoSelector   `json:"repo_selector,omitempty"`
	PeerSelector    *PeerSelector   `json:"peer_selector,omitempty"`
	ReplicationMode *string         `json:"replication_mode,omitempty"`
	Priority        *int64          `json:"priority,omitempty"`
	ArtifactFilter  *ArtifactFilter `json:"artifact_filter,omitempty"`
	Precedence      *int64          `json:"precedence,omitempty"`
}

func (c *Client) CreateSyncPolicy(ctx context.Context, req SyncPolicyRequest) (*SyncPolicy, error) {
	var out SyncPolicy
	if err := c.do(ctx, http.MethodPost, "/sync-policies", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSyncPolicy(ctx context.Context, id string) (*SyncPolicy, error) {
	var out SyncPolicy
	if err := c.do(ctx, http.MethodGet, "/sync-policies/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSyncPolicy(ctx context.Context, id string, req SyncPolicyRequest) (*SyncPolicy, error) {
	var out SyncPolicy
	if err := c.do(ctx, http.MethodPut, "/sync-policies/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSyncPolicy(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/sync-policies/"+url.PathEscape(id), nil, nil)
}
