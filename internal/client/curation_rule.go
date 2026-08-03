package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// CurationRule mirrors the API's RuleResponse (GET /curation/rules/{id}).
type CurationRule struct {
	ID                string  `json:"id"`
	StagingRepoID     *string `json:"staging_repo_id"`
	PackagePattern    string  `json:"package_pattern"`
	VersionConstraint string  `json:"version_constraint"`
	Architecture      string  `json:"architecture"`
	Action            string  `json:"action"`
	Priority          int64   `json:"priority"`
	Reason            string  `json:"reason"`
	Enabled           bool    `json:"enabled"`
	// Typed rules (#2947): rule_type selects the evaluation engine and config
	// holds its parameters. scope is repository or global (immutable).
	RuleType  string          `json:"rule_type"`
	Config    json.RawMessage `json:"config"`
	Scope     string          `json:"scope"`
	CreatedBy *string         `json:"created_by"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

// CreateCurationRuleRequest maps the API's CurationCreateRuleRequest
// (POST /curation/rules). Fields the API defaults server-side
// (version_constraint, architecture, priority) are pointers so they can be
// omitted. The API does not accept `enabled` on create (new rules are always
// enabled); toggle it with a follow-up update.
type CreateCurationRuleRequest struct {
	StagingRepoID     *string `json:"staging_repo_id,omitempty"`
	PackagePattern    string  `json:"package_pattern"`
	VersionConstraint *string `json:"version_constraint,omitempty"`
	Architecture      *string `json:"architecture,omitempty"`
	Action            string  `json:"action"`
	Priority          *int64  `json:"priority,omitempty"`
	Reason            string  `json:"reason"`
	// Typed rules (#2947). rule_type defaults to "pattern" server-side; scope
	// defaults to "repository" and is immutable after create.
	RuleType *string         `json:"rule_type,omitempty"`
	Config   json.RawMessage `json:"config,omitempty"`
	Scope    *string         `json:"scope,omitempty"`
}

// UpdateCurationRuleRequest maps the API's CurationUpdateRuleRequest
// (PUT /curation/rules/{id}). The API replaces the full rule, so every mutable
// field is always sent. staging_repo_id and scope are immutable and cannot be
// changed. rule_type and config MUST always be sent: the backend defaults them
// to "pattern"/{} when omitted, so omitting them would silently reset a typed
// rule to a pattern rule with empty config.
type UpdateCurationRuleRequest struct {
	PackagePattern    string          `json:"package_pattern"`
	VersionConstraint string          `json:"version_constraint"`
	Architecture      string          `json:"architecture"`
	Action            string          `json:"action"`
	Priority          int64           `json:"priority"`
	Reason            string          `json:"reason"`
	Enabled           bool            `json:"enabled"`
	RuleType          string          `json:"rule_type"`
	Config            json.RawMessage `json:"config"`
}

func (c *Client) CreateCurationRule(ctx context.Context, req CreateCurationRuleRequest) (*CurationRule, error) {
	var out CurationRule
	if err := c.do(ctx, http.MethodPost, "/curation/rules", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetCurationRule(ctx context.Context, id string) (*CurationRule, error) {
	var out CurationRule
	if err := c.do(ctx, http.MethodGet, "/curation/rules/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateCurationRule(ctx context.Context, id string, req UpdateCurationRuleRequest) (*CurationRule, error) {
	var out CurationRule
	if err := c.do(ctx, http.MethodPut, "/curation/rules/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteCurationRule(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/curation/rules/"+url.PathEscape(id), nil, nil)
}
