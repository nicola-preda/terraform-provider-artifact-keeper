package client

import (
	"context"
	"net/http"
	"net/url"
)

// RoutingRule is one rewrite rule in a repository's ordered routing list: a
// proxy request whose path matches path_pattern is rewritten to rewrite_to.
type RoutingRule struct {
	PathPattern string `json:"path_pattern"`
	RewriteTo   string `json:"rewrite_to"`
}

// RoutingRulesResponse mirrors the GET body: the repository key plus its
// ordered rules.
type RoutingRulesResponse struct {
	RepositoryKey string        `json:"repository_key"`
	Rules         []RoutingRule `json:"rules"`
}

// SetRoutingRulesRequest maps SetRoutingRulesRequest: the full ordered rule
// list. The POST replaces the whole set.
type SetRoutingRulesRequest struct {
	Rules []RoutingRule `json:"rules"`
}

// GetRepositoryRoutingRules reads GET /repositories/{key}/routing-rules. A
// missing repository surfaces as a 404 APIError.
func (c *Client) GetRepositoryRoutingRules(ctx context.Context, repoKey string) (*RoutingRulesResponse, error) {
	var out RoutingRulesResponse
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repoKey)+"/routing-rules", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetRepositoryRoutingRules replaces the whole ordered rule list via
// POST /repositories/{key}/routing-rules.
func (c *Client) SetRepositoryRoutingRules(ctx context.Context, repoKey string, rules []RoutingRule) error {
	body := SetRoutingRulesRequest{Rules: rules}
	return c.do(ctx, http.MethodPost, "/repositories/"+url.PathEscape(repoKey)+"/routing-rules", body, nil)
}

// DeleteRepositoryRoutingRules clears all rules via
// DELETE /repositories/{key}/routing-rules.
func (c *Client) DeleteRepositoryRoutingRules(ctx context.Context, repoKey string) error {
	return c.do(ctx, http.MethodDelete, "/repositories/"+url.PathEscape(repoKey)+"/routing-rules", nil, nil)
}
