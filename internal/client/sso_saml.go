package client

import (
	"context"
	"net/http"
	"net/url"
)

// SamlConfig mirrors SamlConfigResponse. certificate is never returned.
type SamlConfig struct {
	ID                      string            `json:"id"`
	Name                    string            `json:"name"`
	EntityID                string            `json:"entity_id"`
	SsoURL                  string            `json:"sso_url"`
	SloURL                  *string           `json:"slo_url"`
	HasCertificate          bool              `json:"has_certificate"`
	NameIDFormat            string            `json:"name_id_format"`
	AttributeMapping        map[string]string `json:"attribute_mapping"`
	SpEntityID              string            `json:"sp_entity_id"`
	SignRequests            bool              `json:"sign_requests"`
	RequireSignedAssertions bool              `json:"require_signed_assertions"`
	AdminGroup              *string           `json:"admin_group"`
	IsEnabled               bool              `json:"is_enabled"`
	CreatedAt               string            `json:"created_at"`
	UpdatedAt               string            `json:"updated_at"`
}

// SamlConfigRequest is used for create (POST) and update (PUT).
type SamlConfigRequest struct {
	Name                    *string           `json:"name,omitempty"`
	EntityID                *string           `json:"entity_id,omitempty"`
	SsoURL                  *string           `json:"sso_url,omitempty"`
	SloURL                  *string           `json:"slo_url,omitempty"`
	Certificate             *string           `json:"certificate,omitempty"`
	NameIDFormat            *string           `json:"name_id_format,omitempty"`
	AttributeMapping        map[string]string `json:"attribute_mapping,omitempty"`
	SpEntityID              *string           `json:"sp_entity_id,omitempty"`
	SignRequests            *bool             `json:"sign_requests,omitempty"`
	RequireSignedAssertions *bool             `json:"require_signed_assertions,omitempty"`
	AdminGroup              *string           `json:"admin_group,omitempty"`
	IsEnabled               *bool             `json:"is_enabled,omitempty"`
}

func (c *Client) CreateSamlConfig(ctx context.Context, req SamlConfigRequest) (*SamlConfig, error) {
	var out SamlConfig
	if err := c.do(ctx, http.MethodPost, "/admin/sso/saml", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSamlConfig(ctx context.Context, id string) (*SamlConfig, error) {
	var out SamlConfig
	if err := c.do(ctx, http.MethodGet, "/admin/sso/saml/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSamlConfig(ctx context.Context, id string, req SamlConfigRequest) (*SamlConfig, error) {
	var out SamlConfig
	if err := c.do(ctx, http.MethodPut, "/admin/sso/saml/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSamlConfig(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/admin/sso/saml/"+url.PathEscape(id), nil, nil)
}
