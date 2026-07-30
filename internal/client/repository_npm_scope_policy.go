package client

import (
	"context"
	"net/http"
	"net/url"
)

// NpmScopePolicy is the per-repository npm scope allowlist at
// GET/PUT /repositories/{key}/npm-scope-policy. One per repo (a singleton, no
// delete). Only meaningful for a remote member of an npm virtual repository; the
// backend rejects it for anything else.
type NpmScopePolicy struct {
	RepositoryKey string   `json:"repository_key"`
	AllowedScopes []string `json:"allowed_scopes"`
	AllowUnscoped bool     `json:"allow_unscoped"`
	Active        bool     `json:"active"`
}

// SetNpmScopePolicyRequest maps SetNpmScopePolicyRequest: the scopes to allow
// (each starting with `@`) and whether unscoped packages are permitted. Both
// fields are always sent; the PUT replaces the stored policy.
type SetNpmScopePolicyRequest struct {
	AllowedScopes []string `json:"allowed_scopes"`
	AllowUnscoped bool     `json:"allow_unscoped"`
}

// GetNpmScopePolicy reads GET /repositories/{key}/npm-scope-policy.
func (c *Client) GetNpmScopePolicy(ctx context.Context, repoKey string) (*NpmScopePolicy, error) {
	var out NpmScopePolicy
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repoKey)+"/npm-scope-policy", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetNpmScopePolicy replaces the policy via
// PUT /repositories/{key}/npm-scope-policy. Read the result back with
// GetNpmScopePolicy.
func (c *Client) SetNpmScopePolicy(ctx context.Context, repoKey string, req SetNpmScopePolicyRequest) error {
	return c.do(ctx, http.MethodPut, "/repositories/"+url.PathEscape(repoKey)+"/npm-scope-policy", req, nil)
}
