package client

import (
	"context"
	"net/http"
	"net/url"
)

// UpstreamAuthRequest maps the PUT /repositories/{key}/upstream-auth body: the
// credentials a remote repository uses to authenticate to its upstream.
// auth_type is one of "basic", "bearer", "none" ("none" removes the auth).
// password carries the basic password or the bearer token; it is never returned.
type UpstreamAuthRequest struct {
	AuthType string  `json:"auth_type"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

// SetUpstreamAuth writes the upstream credentials via PUT. Write-only: there is
// no GET, and the response carries nothing we keep, so it is discarded.
func (c *Client) SetUpstreamAuth(ctx context.Context, repoKey string, req UpstreamAuthRequest) error {
	return c.do(ctx, http.MethodPut, "/repositories/"+url.PathEscape(repoKey)+"/upstream-auth", req, nil)
}
