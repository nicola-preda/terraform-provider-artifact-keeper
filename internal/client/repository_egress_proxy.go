package client

import (
	"context"
	"net/http"
	"net/url"
)

// EgressProxy is a remote repository's outbound proxy config: the body of
// GET/PUT /repositories/{key}/egress-proxy (1.8.0, #2469/#2811). A singleton
// per repository, remote repositories only, repo-admin gated on both verbs.
//
// A per-repo setting overrides the process-wide HTTP_PROXY environment, it
// never merges with it.
type EgressProxy struct {
	// Mode is inherit (follow the process environment), direct (bypass it) or
	// explicit (use ProxyURL).
	Mode string `json:"mode"`
	// ProxyURL comes back with any user:pass@ replaced by "***", so a
	// credentialed URL never round-trips. ProxyCredentialsConfigured is the
	// observable half. Absent unless Mode is explicit.
	ProxyURL *string `json:"proxy_url"`
	NoProxy  *string `json:"no_proxy"`
	// ProxyCredentialsConfigured reports whether the stored URL carries
	// userinfo, without disclosing it.
	ProxyCredentialsConfigured bool `json:"proxy_credentials_configured"`
}

// SetEgressProxyRequest maps EgressProxyRequest. ProxyURL is required when Mode
// is explicit and rejected (400) for the other two modes, which also drop
// NoProxy.
type SetEgressProxyRequest struct {
	Mode     string  `json:"mode"`
	ProxyURL *string `json:"proxy_url,omitempty"`
	NoProxy  *string `json:"no_proxy,omitempty"`
}

// GetRepositoryEgressProxy reads GET /repositories/{key}/egress-proxy.
func (c *Client) GetRepositoryEgressProxy(ctx context.Context, repoKey string) (*EgressProxy, error) {
	var out EgressProxy
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repoKey)+"/egress-proxy", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetRepositoryEgressProxy upserts via PUT /repositories/{key}/egress-proxy and
// returns the stored config, so no follow-up GET is needed.
func (c *Client) SetRepositoryEgressProxy(ctx context.Context, repoKey string, req SetEgressProxyRequest) (*EgressProxy, error) {
	var out EgressProxy
	if err := c.do(ctx, http.MethodPut, "/repositories/"+url.PathEscape(repoKey)+"/egress-proxy", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
