package client

import (
	"context"
	"net/http"
	"net/url"
)

// CiOidcProvider mirrors CiOidcProviderResponse. There is no secret: CI OIDC
// trusts JWTs issued by the configured issuer (workload identity), so the
// provider is validated against the issuer's JWKS rather than a client secret.
type CiOidcProvider struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProviderType string `json:"provider_type"`
	IssuerURL    string `json:"issuer_url"`
	Audience     string `json:"audience"`
	IsEnabled    bool   `json:"is_enabled"`
	MappingCount int64  `json:"mapping_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// CiOidcProviderRequest is used for both create (POST) and update (PUT).
// Pointers distinguish "omit" from an explicit value. On create the backend
// fills defaults for omitted fields (provider_type="generic",
// audience="artifact-keeper", is_enabled=true); on update an omitted field
// keeps its existing value.
type CiOidcProviderRequest struct {
	Name         *string `json:"name,omitempty"`
	ProviderType *string `json:"provider_type,omitempty"`
	IssuerURL    *string `json:"issuer_url,omitempty"`
	Audience     *string `json:"audience,omitempty"`
	IsEnabled    *bool   `json:"is_enabled,omitempty"`
}

func (c *Client) CreateCiOidcProvider(ctx context.Context, req CiOidcProviderRequest) (*CiOidcProvider, error) {
	var out CiOidcProvider
	if err := c.do(ctx, http.MethodPost, "/admin/ci-oidc", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetCiOidcProvider(ctx context.Context, id string) (*CiOidcProvider, error) {
	var out CiOidcProvider
	if err := c.do(ctx, http.MethodGet, "/admin/ci-oidc/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateCiOidcProvider(ctx context.Context, id string, req CiOidcProviderRequest) (*CiOidcProvider, error) {
	var out CiOidcProvider
	if err := c.do(ctx, http.MethodPut, "/admin/ci-oidc/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteCiOidcProvider(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/admin/ci-oidc/"+url.PathEscape(id), nil, nil)
}
