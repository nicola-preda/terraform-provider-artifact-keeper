package client

import (
	"context"
	"net/http"
	"net/url"
)

// PromotionRule mirrors PromotionRuleResponse (GET /promotion-rules/{id}). It
// gates auto-promotion of artifacts from a staging source repository to a
// release target repository.
type PromotionRule struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	SourceRepoID       string   `json:"source_repo_id"`
	TargetRepoID       string   `json:"target_repo_id"`
	IsEnabled          bool     `json:"is_enabled"`
	MaxCveSeverity     *string  `json:"max_cve_severity"`
	AllowedLicenses    []string `json:"allowed_licenses"`
	RequireSignature   bool     `json:"require_signature"`
	MinStagingHours    *int64   `json:"min_staging_hours"`
	MaxArtifactAgeDays *int64   `json:"max_artifact_age_days"`
	MinHealthScore     *int64   `json:"min_health_score"`
	AutoPromote        bool     `json:"auto_promote"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

// CreatePromotionRuleRequest maps the API's CreateRuleRequest (POST
// /promotion-rules). Fields with server-side defaults use pointers so an
// omitted value falls back to the API default.
type CreatePromotionRuleRequest struct {
	Name               string   `json:"name"`
	SourceRepoID       string   `json:"source_repo_id"`
	TargetRepoID       string   `json:"target_repo_id"`
	IsEnabled          *bool    `json:"is_enabled,omitempty"`
	MaxCveSeverity     *string  `json:"max_cve_severity,omitempty"`
	AllowedLicenses    []string `json:"allowed_licenses,omitempty"`
	RequireSignature   *bool    `json:"require_signature,omitempty"`
	MinStagingHours    *int64   `json:"min_staging_hours,omitempty"`
	MaxArtifactAgeDays *int64   `json:"max_artifact_age_days,omitempty"`
	MinHealthScore     *int64   `json:"min_health_score,omitempty"`
	AutoPromote        *bool    `json:"auto_promote,omitempty"`
}

// UpdatePromotionRuleRequest maps the API's UpdateRuleRequest (PUT
// /promotion-rules/{id}). Every field is optional; an omitted field preserves
// its current value server-side (source/target repo are immutable and absent).
type UpdatePromotionRuleRequest struct {
	Name               *string  `json:"name,omitempty"`
	IsEnabled          *bool    `json:"is_enabled,omitempty"`
	MaxCveSeverity     *string  `json:"max_cve_severity,omitempty"`
	AllowedLicenses    []string `json:"allowed_licenses,omitempty"`
	RequireSignature   *bool    `json:"require_signature,omitempty"`
	MinStagingHours    *int64   `json:"min_staging_hours,omitempty"`
	MaxArtifactAgeDays *int64   `json:"max_artifact_age_days,omitempty"`
	MinHealthScore     *int64   `json:"min_health_score,omitempty"`
	AutoPromote        *bool    `json:"auto_promote,omitempty"`
}

func (c *Client) CreatePromotionRule(ctx context.Context, req CreatePromotionRuleRequest) (*PromotionRule, error) {
	var out PromotionRule
	if err := c.do(ctx, http.MethodPost, "/promotion-rules", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPromotionRule(ctx context.Context, id string) (*PromotionRule, error) {
	var out PromotionRule
	if err := c.do(ctx, http.MethodGet, "/promotion-rules/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdatePromotionRule(ctx context.Context, id string, req UpdatePromotionRuleRequest) (*PromotionRule, error) {
	var out PromotionRule
	if err := c.do(ctx, http.MethodPut, "/promotion-rules/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeletePromotionRule(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/promotion-rules/"+url.PathEscape(id), nil, nil)
}
