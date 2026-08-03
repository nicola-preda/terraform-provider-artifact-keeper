package client

import (
	"context"
	"net/http"
	"net/url"
)

// RepositorySecurity is the per-repository scan config: the `config` object of
// GET /repositories/{key}/security. One per repo, defaulted when the repo is
// created.
type RepositorySecurity struct {
	ID                     string `json:"id"`
	RepositoryID           string `json:"repository_id"`
	ScanEnabled            bool   `json:"scan_enabled"`
	ScanOnUpload           bool   `json:"scan_on_upload"`
	ScanOnProxy            bool   `json:"scan_on_proxy"`
	BlockOnPolicyViolation bool   `json:"block_on_policy_violation"`
	SeverityThreshold      string `json:"severity_threshold"`
	// Fail-open vs fail-closed for the inline proxy scan-on-fetch (#2954):
	// "fail_open" (default) serves the artifact if the scan cannot run,
	// "fail_closed" refuses it.
	ProxyScanAction string `json:"proxy_scan_action"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// repoSecurityEnvelope wraps the GET body ({config, score}); only config is
// managed here.
type repoSecurityEnvelope struct {
	Config *RepositorySecurity `json:"config"`
}

// UpdateRepositorySecurityRequest maps UpsertScanConfigRequest. Every field is
// optional: the PUT upserts, so an omitted field keeps its current value.
type UpdateRepositorySecurityRequest struct {
	ScanEnabled            *bool   `json:"scan_enabled,omitempty"`
	ScanOnUpload           *bool   `json:"scan_on_upload,omitempty"`
	ScanOnProxy            *bool   `json:"scan_on_proxy,omitempty"`
	BlockOnPolicyViolation *bool   `json:"block_on_policy_violation,omitempty"`
	SeverityThreshold      *string `json:"severity_threshold,omitempty"`
	ProxyScanAction        *string `json:"proxy_scan_action,omitempty"`
}

// GetRepositorySecurity reads GET /repositories/{key}/security and returns its
// `config` (nil when the repo has no scan config).
func (c *Client) GetRepositorySecurity(ctx context.Context, repoKey string) (*RepositorySecurity, error) {
	var out repoSecurityEnvelope
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repoKey)+"/security", nil, &out); err != nil {
		return nil, err
	}
	return out.Config, nil
}

// UpdateRepositorySecurity upserts via PUT /repositories/{key}/security. Admin
// only. Read the result back with GetRepositorySecurity.
func (c *Client) UpdateRepositorySecurity(ctx context.Context, repoKey string, req UpdateRepositorySecurityRequest) error {
	return c.do(ctx, http.MethodPut, "/repositories/"+url.PathEscape(repoKey)+"/security", req, nil)
}
