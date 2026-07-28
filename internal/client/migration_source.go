package client

import (
	"context"
	"net/http"
	"net/url"
)

// MigrationSource mirrors ConnectionResponse (a persistent migration source
// connection, e.g. a Nexus or Artifactory instance). Credentials are never
// returned.
type MigrationSource struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	URL        string  `json:"url"`
	AuthType   string  `json:"auth_type"`
	SourceType string  `json:"source_type"`
	CreatedAt  string  `json:"created_at"`
	VerifiedAt *string `json:"verified_at"`
}

// MigrationCredentials is the nested credentials object on create.
type MigrationCredentials struct {
	Token    *string `json:"token,omitempty"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

// CreateMigrationSourceRequest maps CreateConnectionRequest.
type CreateMigrationSourceRequest struct {
	Name        string               `json:"name"`
	URL         string               `json:"url"`
	AuthType    string               `json:"auth_type"`
	Credentials MigrationCredentials `json:"credentials"`
	SourceType  *string              `json:"source_type,omitempty"`
}

func (c *Client) CreateMigrationSource(ctx context.Context, req CreateMigrationSourceRequest) (*MigrationSource, error) {
	var out MigrationSource
	if err := c.do(ctx, http.MethodPost, "/migrations/connections", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetMigrationSource(ctx context.Context, id string) (*MigrationSource, error) {
	var out MigrationSource
	if err := c.do(ctx, http.MethodGet, "/migrations/connections/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteMigrationSource(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/migrations/connections/"+url.PathEscape(id), nil, nil)
}
