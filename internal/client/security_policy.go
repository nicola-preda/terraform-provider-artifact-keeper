package client

import (
	"context"
	"net/http"
	"net/url"
)

// SecurityPolicy mirrors PolicyResponse (a scan/promotion gating policy under
// GET /security/policies/{id}).
type SecurityPolicy struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	RepositoryID       *string `json:"repository_id"`
	MaxSeverity        string  `json:"max_severity"`
	BlockUnscanned     bool    `json:"block_unscanned"`
	BlockOnFail        bool    `json:"block_on_fail"`
	IsEnabled          bool    `json:"is_enabled"`
	MinStagingHours    *int64  `json:"min_staging_hours"`
	MaxArtifactAgeDays *int64  `json:"max_artifact_age_days"`
	RequireSignature   bool    `json:"require_signature"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// CreateSecurityPolicyRequest maps CreatePolicyRequest. block_unscanned and
// require_signature carry server-side defaults, so they are pointers and are
// omitted when unset; is_enabled is not accepted on create.
type CreateSecurityPolicyRequest struct {
	Name               string  `json:"name"`
	RepositoryID       *string `json:"repository_id,omitempty"`
	MaxSeverity        string  `json:"max_severity"`
	BlockUnscanned     *bool   `json:"block_unscanned,omitempty"`
	BlockOnFail        bool    `json:"block_on_fail"`
	MinStagingHours    *int64  `json:"min_staging_hours,omitempty"`
	MaxArtifactAgeDays *int64  `json:"max_artifact_age_days,omitempty"`
	RequireSignature   *bool   `json:"require_signature,omitempty"`
}

// UpdateSecurityPolicyRequest maps UpdatePolicyRequest. Every field is optional;
// an omitted field leaves the stored value untouched (COALESCE update). The API
// does not support clearing min_staging_hours / max_artifact_age_days back to
// null, nor changing repository_id.
type UpdateSecurityPolicyRequest struct {
	Name               *string `json:"name,omitempty"`
	MaxSeverity        *string `json:"max_severity,omitempty"`
	BlockUnscanned     *bool   `json:"block_unscanned,omitempty"`
	BlockOnFail        *bool   `json:"block_on_fail,omitempty"`
	IsEnabled          *bool   `json:"is_enabled,omitempty"`
	MinStagingHours    *int64  `json:"min_staging_hours,omitempty"`
	MaxArtifactAgeDays *int64  `json:"max_artifact_age_days,omitempty"`
	RequireSignature   *bool   `json:"require_signature,omitempty"`
}

func (c *Client) CreateSecurityPolicy(ctx context.Context, req CreateSecurityPolicyRequest) (*SecurityPolicy, error) {
	var out SecurityPolicy
	if err := c.do(ctx, http.MethodPost, "/security/policies", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSecurityPolicy(ctx context.Context, id string) (*SecurityPolicy, error) {
	var out SecurityPolicy
	if err := c.do(ctx, http.MethodGet, "/security/policies/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSecurityPolicy(ctx context.Context, id string, req UpdateSecurityPolicyRequest) (*SecurityPolicy, error) {
	var out SecurityPolicy
	if err := c.do(ctx, http.MethodPut, "/security/policies/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSecurityPolicy(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/security/policies/"+url.PathEscape(id), nil, nil)
}
