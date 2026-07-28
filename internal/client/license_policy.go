package client

import (
	"context"
	"net/http"
	"net/url"
)

// LicensePolicy mirrors LicensePolicyResponse: global (no repository_id) or
// scoped to one repository. No update endpoint; POST is an upsert, so mutable
// fields are treated as immutable.
type LicensePolicy struct {
	ID              string   `json:"id"`
	RepositoryID    *string  `json:"repository_id"`
	Name            string   `json:"name"`
	Description     *string  `json:"description"`
	AllowedLicenses []string `json:"allowed_licenses"`
	DeniedLicenses  []string `json:"denied_licenses"`
	AllowUnknown    bool     `json:"allow_unknown"`
	Action          string   `json:"action"`
	IsEnabled       bool     `json:"is_enabled"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       *string  `json:"updated_at"`
}

// UpsertLicensePolicyRequest maps UpsertLicensePolicyRequest. allowed_licenses
// and denied_licenses are required: send a non-nil slice, never null.
// allow_unknown, action, and is_enabled have server-side defaults.
type UpsertLicensePolicyRequest struct {
	RepositoryID    *string  `json:"repository_id,omitempty"`
	Name            string   `json:"name"`
	Description     *string  `json:"description,omitempty"`
	AllowedLicenses []string `json:"allowed_licenses"`
	DeniedLicenses  []string `json:"denied_licenses"`
	AllowUnknown    *bool    `json:"allow_unknown,omitempty"`
	Action          *string  `json:"action,omitempty"`
	IsEnabled       *bool    `json:"is_enabled,omitempty"`
}

func (c *Client) CreateLicensePolicy(ctx context.Context, req UpsertLicensePolicyRequest) (*LicensePolicy, error) {
	var out LicensePolicy
	if err := c.do(ctx, http.MethodPost, "/sbom/license-policies", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetLicensePolicy(ctx context.Context, id string) (*LicensePolicy, error) {
	var out LicensePolicy
	if err := c.do(ctx, http.MethodGet, "/sbom/license-policies/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteLicensePolicy(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/sbom/license-policies/"+url.PathEscape(id), nil, nil)
}
