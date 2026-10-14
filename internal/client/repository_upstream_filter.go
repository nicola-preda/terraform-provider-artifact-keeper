package client

import (
	"context"
	"net/http"
	"net/url"
)

// UpstreamFilter is the body of PUT /repositories/{key}/upstream-filter
// (1.11.0, #840): which paths a Remote repository may request upstream.
type UpstreamFilter struct {
	IncludePatterns []string `json:"include_patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
}

// UpstreamFilterResponse mirrors UpstreamFilterResponse.
type UpstreamFilterResponse struct {
	IncludePatterns []string `json:"include_patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
	// Active is false when no filter is configured.
	Active bool `json:"active"`
	// Error is set when the stored filter is unusable and every fetch is refused.
	Error *string `json:"error"`
}

func upstreamFilterPath(key string) string {
	return "/repositories/" + url.PathEscape(key) + "/upstream-filter"
}

func (c *Client) GetUpstreamFilter(ctx context.Context, key string) (*UpstreamFilterResponse, error) {
	var out UpstreamFilterResponse
	if err := c.do(ctx, http.MethodGet, upstreamFilterPath(key), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetUpstreamFilter replaces the filter. Two empty lists remove it.
func (c *Client) SetUpstreamFilter(ctx context.Context, key string, req UpstreamFilter) (*UpstreamFilterResponse, error) {
	var out UpstreamFilterResponse
	if err := c.do(ctx, http.MethodPut, upstreamFilterPath(key), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteUpstreamFilter(ctx context.Context, key string) error {
	return c.do(ctx, http.MethodDelete, upstreamFilterPath(key), nil, nil)
}
