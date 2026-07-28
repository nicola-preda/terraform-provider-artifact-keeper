package client

import (
	"context"
	"net/http"
	"net/url"
)

// QualityGate mirrors the API's GateResponse (GET /quality/gates/{id}).
type QualityGate struct {
	ID                 string   `json:"id"`
	RepositoryID       *string  `json:"repository_id"`
	Name               string   `json:"name"`
	Description        *string  `json:"description"`
	MinHealthScore     *int64   `json:"min_health_score"`
	MinSecurityScore   *int64   `json:"min_security_score"`
	MinQualityScore    *int64   `json:"min_quality_score"`
	MinMetadataScore   *int64   `json:"min_metadata_score"`
	MaxCriticalIssues  *int64   `json:"max_critical_issues"`
	MaxHighIssues      *int64   `json:"max_high_issues"`
	MaxMediumIssues    *int64   `json:"max_medium_issues"`
	RequiredChecks     []string `json:"required_checks"`
	EnforceOnPromotion bool     `json:"enforce_on_promotion"`
	EnforceOnDownload  bool     `json:"enforce_on_download"`
	Action             string   `json:"action"`
	IsEnabled          bool     `json:"is_enabled"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

// CreateQualityGateRequest maps the API's CreateGateRequest (POST /quality/gates).
// Fields the API defaults server-side (required_checks, enforce_on_promotion,
// enforce_on_download, action) are omitempty. The API does not accept
// `is_enabled` on create (new gates are always enabled); toggle it with a
// follow-up update.
type CreateQualityGateRequest struct {
	RepositoryID       *string  `json:"repository_id,omitempty"`
	Name               string   `json:"name"`
	Description        *string  `json:"description,omitempty"`
	MinHealthScore     *int64   `json:"min_health_score,omitempty"`
	MinSecurityScore   *int64   `json:"min_security_score,omitempty"`
	MinQualityScore    *int64   `json:"min_quality_score,omitempty"`
	MinMetadataScore   *int64   `json:"min_metadata_score,omitempty"`
	MaxCriticalIssues  *int64   `json:"max_critical_issues,omitempty"`
	MaxHighIssues      *int64   `json:"max_high_issues,omitempty"`
	MaxMediumIssues    *int64   `json:"max_medium_issues,omitempty"`
	RequiredChecks     []string `json:"required_checks,omitempty"`
	EnforceOnPromotion *bool    `json:"enforce_on_promotion,omitempty"`
	EnforceOnDownload  *bool    `json:"enforce_on_download,omitempty"`
	Action             *string  `json:"action,omitempty"`
}

// UpdateQualityGateRequest maps the API's UpdateGateRequest (PUT /quality/gates/{id}).
// The API applies a partial update: any omitted field keeps its current value.
// repository_id is immutable and cannot be changed.
type UpdateQualityGateRequest struct {
	Name               *string   `json:"name,omitempty"`
	Description        *string   `json:"description,omitempty"`
	MinHealthScore     *int64    `json:"min_health_score,omitempty"`
	MinSecurityScore   *int64    `json:"min_security_score,omitempty"`
	MinQualityScore    *int64    `json:"min_quality_score,omitempty"`
	MinMetadataScore   *int64    `json:"min_metadata_score,omitempty"`
	MaxCriticalIssues  *int64    `json:"max_critical_issues,omitempty"`
	MaxHighIssues      *int64    `json:"max_high_issues,omitempty"`
	MaxMediumIssues    *int64    `json:"max_medium_issues,omitempty"`
	RequiredChecks     *[]string `json:"required_checks,omitempty"`
	EnforceOnPromotion *bool     `json:"enforce_on_promotion,omitempty"`
	EnforceOnDownload  *bool     `json:"enforce_on_download,omitempty"`
	Action             *string   `json:"action,omitempty"`
	IsEnabled          *bool     `json:"is_enabled,omitempty"`
}

func (c *Client) CreateQualityGate(ctx context.Context, req CreateQualityGateRequest) (*QualityGate, error) {
	var out QualityGate
	if err := c.do(ctx, http.MethodPost, "/quality/gates", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetQualityGate(ctx context.Context, id string) (*QualityGate, error) {
	var out QualityGate
	if err := c.do(ctx, http.MethodGet, "/quality/gates/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateQualityGate(ctx context.Context, id string, req UpdateQualityGateRequest) (*QualityGate, error) {
	var out QualityGate
	if err := c.do(ctx, http.MethodPut, "/quality/gates/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteQualityGate(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/quality/gates/"+url.PathEscape(id), nil, nil)
}
