package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// LifecyclePolicy mirrors the API's LifecyclePolicy (GET /admin/lifecycle/{id}).
type LifecyclePolicy struct {
	ID                  string          `json:"id"`
	RepositoryID        *string         `json:"repository_id"`
	Name                string          `json:"name"`
	Description         *string         `json:"description"`
	Enabled             bool            `json:"enabled"`
	PolicyType          string          `json:"policy_type"`
	Config              json.RawMessage `json:"config"`
	Priority            int64           `json:"priority"`
	LastRunAt           *string         `json:"last_run_at"`
	LastRunItemsRemoved *int64          `json:"last_run_items_removed"`
	CronSchedule        *string         `json:"cron_schedule"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
}

// CreateLifecyclePolicyRequest maps the API's CreateLifecyclePolicyRequest
// (POST /admin/lifecycle). The API does not accept `enabled` on create (new
// policies are always enabled); toggle it with a follow-up update.
type CreateLifecyclePolicyRequest struct {
	RepositoryID *string         `json:"repository_id,omitempty"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	PolicyType   string          `json:"policy_type"`
	Config       json.RawMessage `json:"config"`
	Priority     *int64          `json:"priority,omitempty"`
	CronSchedule *string         `json:"cron_schedule,omitempty"`
}

// UpdateLifecyclePolicyRequest maps the API's UpdateLifecyclePolicyRequest
// (PATCH /admin/lifecycle/{id}). The API applies a partial update: any omitted
// field keeps its current value. repository_id and policy_type are immutable.
type UpdateLifecyclePolicyRequest struct {
	Name         *string         `json:"name,omitempty"`
	Description  *string         `json:"description,omitempty"`
	Enabled      *bool           `json:"enabled,omitempty"`
	Config       json.RawMessage `json:"config,omitempty"`
	Priority     *int64          `json:"priority,omitempty"`
	CronSchedule *string         `json:"cron_schedule,omitempty"`
}

func (c *Client) CreateLifecyclePolicy(ctx context.Context, req CreateLifecyclePolicyRequest) (*LifecyclePolicy, error) {
	var out LifecyclePolicy
	if err := c.do(ctx, http.MethodPost, "/admin/lifecycle", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetLifecyclePolicy(ctx context.Context, id string) (*LifecyclePolicy, error) {
	var out LifecyclePolicy
	if err := c.do(ctx, http.MethodGet, "/admin/lifecycle/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateLifecyclePolicy(ctx context.Context, id string, req UpdateLifecyclePolicyRequest) (*LifecyclePolicy, error) {
	var out LifecyclePolicy
	if err := c.do(ctx, http.MethodPatch, "/admin/lifecycle/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteLifecyclePolicy(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/admin/lifecycle/"+url.PathEscape(id), nil, nil)
}
