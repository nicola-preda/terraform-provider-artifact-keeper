package client

import (
	"context"
	"net/http"
	"net/url"
)

// OidcConfig mirrors OidcConfigResponse. client_secret is never returned;
// has_secret indicates whether one is set.
type OidcConfig struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	IssuerURL         string            `json:"issuer_url"`
	ClientID          string            `json:"client_id"`
	HasSecret         bool              `json:"has_secret"`
	Scopes            []string          `json:"scopes"`
	AttributeMapping  map[string]string `json:"attribute_mapping"`
	IsEnabled         bool              `json:"is_enabled"`
	AutoCreateUsers   bool              `json:"auto_create_users"`
	PkceEnabled       bool              `json:"pkce_enabled"`
	MapGroupsToGroups bool              `json:"map_groups_to_groups"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
}

// OidcConfigRequest is used for both create (POST) and update (PUT). Pointers
// distinguish "omit" from "set to zero value".
type OidcConfigRequest struct {
	Name              *string           `json:"name,omitempty"`
	IssuerURL         *string           `json:"issuer_url,omitempty"`
	ClientID          *string           `json:"client_id,omitempty"`
	ClientSecret      *string           `json:"client_secret,omitempty"`
	Scopes            []string          `json:"scopes,omitempty"`
	AttributeMapping  map[string]string `json:"attribute_mapping,omitempty"`
	IsEnabled         *bool             `json:"is_enabled,omitempty"`
	AutoCreateUsers   *bool             `json:"auto_create_users,omitempty"`
	PkceEnabled       *bool             `json:"pkce_enabled,omitempty"`
	MapGroupsToGroups *bool             `json:"map_groups_to_groups,omitempty"`
}

func (c *Client) CreateOidcConfig(ctx context.Context, req OidcConfigRequest) (*OidcConfig, error) {
	var out OidcConfig
	if err := c.do(ctx, http.MethodPost, "/admin/sso/oidc", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetOidcConfig(ctx context.Context, id string) (*OidcConfig, error) {
	var out OidcConfig
	if err := c.do(ctx, http.MethodGet, "/admin/sso/oidc/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateOidcConfig(ctx context.Context, id string, req OidcConfigRequest) (*OidcConfig, error) {
	var out OidcConfig
	if err := c.do(ctx, http.MethodPut, "/admin/sso/oidc/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteOidcConfig(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/admin/sso/oidc/"+url.PathEscape(id), nil, nil)
}
