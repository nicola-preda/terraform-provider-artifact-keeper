package client

import (
	"context"
	"net/http"
	"net/url"
)

// AgeGateConfig mirrors AgeGateConfigResponse for a single repository's
// age-gate (age-based proxy quality gate). It is a per-repository singleton
// addressed by repository key; it has no create or delete endpoint, only
// GET and PUT.
type AgeGateConfig struct {
	RepositoryKey string `json:"repository_key"`
	Enabled       bool   `json:"enabled"`
	MinAgeDays    int64  `json:"min_age_days"`
}

// AgeGateConfigRequest maps UpdateAgeGateConfigRequest (PUT). Both fields are
// mandatory on the backend.
type AgeGateConfigRequest struct {
	Enabled    bool  `json:"enabled"`
	MinAgeDays int64 `json:"min_age_days"`
}

func (c *Client) GetAgeGateConfig(ctx context.Context, repositoryKey string) (*AgeGateConfig, error) {
	var out AgeGateConfig
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repositoryKey)+"/age-gate", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateAgeGateConfig(ctx context.Context, repositoryKey string, req AgeGateConfigRequest) (*AgeGateConfig, error) {
	var out AgeGateConfig
	if err := c.do(ctx, http.MethodPut, "/repositories/"+url.PathEscape(repositoryKey)+"/age-gate", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
