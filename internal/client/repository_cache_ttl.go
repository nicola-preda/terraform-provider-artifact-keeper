package client

import (
	"context"
	"net/http"
	"net/url"
)

// CacheTtlResponse is the per-repository cache TTL: the body of
// GET /repositories/{key}/cache-ttl. One per repo, a singleton.
type CacheTtlResponse struct {
	RepositoryKey   string `json:"repository_key"`
	CacheTtlSeconds int64  `json:"cache_ttl_seconds"`
}

// SetCacheTtlRequest maps SetCacheTtlRequest: the PUT payload.
type SetCacheTtlRequest struct {
	CacheTtlSeconds int64 `json:"cache_ttl_seconds"`
}

// GetRepositoryCacheTtl reads GET /repositories/{key}/cache-ttl.
func (c *Client) GetRepositoryCacheTtl(ctx context.Context, repoKey string) (*CacheTtlResponse, error) {
	var out CacheTtlResponse
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repoKey)+"/cache-ttl", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetRepositoryCacheTtl upserts via PUT /repositories/{key}/cache-ttl. Read the
// result back with GetRepositoryCacheTtl.
func (c *Client) SetRepositoryCacheTtl(ctx context.Context, repoKey string, req SetCacheTtlRequest) error {
	return c.do(ctx, http.MethodPut, "/repositories/"+url.PathEscape(repoKey)+"/cache-ttl", req, nil)
}
