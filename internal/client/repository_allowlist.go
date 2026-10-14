package client

import (
	"context"
	"net/http"
	"net/url"
)

// AllowlistEntry is one admitted package of a virtual conda channel's
// allowlist (1.11.0). The backend rejects unknown fields.
type AllowlistEntry struct {
	Name    string   `json:"name"`
	Version *string  `json:"version,omitempty"`
	Subdirs []string `json:"subdirs,omitempty"`
}

// RepositoryAllowlist is the body of PUT /repositories/{key}/allowlist.
type RepositoryAllowlist struct {
	Enabled bool             `json:"enabled"`
	Entries []AllowlistEntry `json:"entries"`
}

// RepositoryAllowlistResponse mirrors CondaAllowlistResponse.
type RepositoryAllowlistResponse struct {
	Enabled bool             `json:"enabled"`
	Entries []AllowlistEntry `json:"entries"`
	Error   *string          `json:"error"`
}

func allowlistPath(key string) string {
	return "/repositories/" + url.PathEscape(key) + "/allowlist"
}

func (c *Client) GetRepositoryAllowlist(ctx context.Context, key string) (*RepositoryAllowlistResponse, error) {
	var out RepositoryAllowlistResponse
	if err := c.do(ctx, http.MethodGet, allowlistPath(key), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SetRepositoryAllowlist(ctx context.Context, key string, req RepositoryAllowlist) (*RepositoryAllowlistResponse, error) {
	var out RepositoryAllowlistResponse
	if err := c.do(ctx, http.MethodPut, allowlistPath(key), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRepositoryAllowlist(ctx context.Context, key string) error {
	return c.do(ctx, http.MethodDelete, allowlistPath(key), nil, nil)
}
