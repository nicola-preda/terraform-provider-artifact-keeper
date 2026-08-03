package client

import (
	"context"
	"net/http"
	"net/url"
)

// LdapConfig mirrors LdapConfigResponse. bind_password is never returned.
type LdapConfig struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	ServerURL            string  `json:"server_url"`
	BindDN               *string `json:"bind_dn"`
	HasBindPassword      bool    `json:"has_bind_password"`
	UserBaseDN           string  `json:"user_base_dn"`
	UserFilter           string  `json:"user_filter"`
	GroupBaseDN          *string `json:"group_base_dn"`
	GroupFilter          *string `json:"group_filter"`
	EmailAttribute       string  `json:"email_attribute"`
	DisplayNameAttribute string  `json:"display_name_attribute"`
	UsernameAttribute    string  `json:"username_attribute"`
	GroupsAttribute      string  `json:"groups_attribute"`
	AdminGroupDN         *string `json:"admin_group_dn"`
	UseStartTLS          bool    `json:"use_starttls"`
	InsecureSkipVerify   bool    `json:"insecure_skip_verify"`
	HasCaCertificate     bool    `json:"has_ca_certificate"`
	IsEnabled            bool    `json:"is_enabled"`
	Priority             int64   `json:"priority"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

// LdapConfigRequest is used for create (POST) and update (PUT).
type LdapConfigRequest struct {
	Name                 *string `json:"name,omitempty"`
	ServerURL            *string `json:"server_url,omitempty"`
	BindDN               *string `json:"bind_dn,omitempty"`
	BindPassword         *string `json:"bind_password,omitempty"`
	UserBaseDN           *string `json:"user_base_dn,omitempty"`
	UserFilter           *string `json:"user_filter,omitempty"`
	GroupBaseDN          *string `json:"group_base_dn,omitempty"`
	GroupFilter          *string `json:"group_filter,omitempty"`
	EmailAttribute       *string `json:"email_attribute,omitempty"`
	DisplayNameAttribute *string `json:"display_name_attribute,omitempty"`
	UsernameAttribute    *string `json:"username_attribute,omitempty"`
	GroupsAttribute      *string `json:"groups_attribute,omitempty"`
	AdminGroupDN         *string `json:"admin_group_dn,omitempty"`
	UseStartTLS          *bool   `json:"use_starttls,omitempty"`
	// InsecureSkipVerify disables TLS certificate verification of the LDAP server.
	InsecureSkipVerify *bool `json:"insecure_skip_verify,omitempty"`
	// CaCertificate is a PEM CA bundle used to verify the LDAP server. Write-only,
	// never returned (the response exposes only has_ca_certificate). Send an empty
	// string to clear it.
	CaCertificate *string `json:"ca_certificate,omitempty"`
	IsEnabled     *bool   `json:"is_enabled,omitempty"`
	Priority      *int64  `json:"priority,omitempty"`
}

func (c *Client) CreateLdapConfig(ctx context.Context, req LdapConfigRequest) (*LdapConfig, error) {
	var out LdapConfig
	if err := c.do(ctx, http.MethodPost, "/admin/sso/ldap", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetLdapConfig(ctx context.Context, id string) (*LdapConfig, error) {
	var out LdapConfig
	if err := c.do(ctx, http.MethodGet, "/admin/sso/ldap/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateLdapConfig(ctx context.Context, id string, req LdapConfigRequest) (*LdapConfig, error) {
	var out LdapConfig
	if err := c.do(ctx, http.MethodPut, "/admin/sso/ldap/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteLdapConfig(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/admin/sso/ldap/"+url.PathEscape(id), nil, nil)
}
