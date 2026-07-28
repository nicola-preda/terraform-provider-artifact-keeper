package client

import (
	"context"
	"net/http"
	"net/url"
)

// ServiceAccount mirrors the API's ServiceAccountResponse. A service account is
// a machine identity; the API derives its immutable username from the create
// `name` (prefixed with "svc-") and never returns that original name.
type ServiceAccount struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name"`
	IsActive    bool    `json:"is_active"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// CreateServiceAccountRequest maps the API's CreateServiceAccountRequest. The
// `description` field becomes the account's display_name.
type CreateServiceAccountRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// UpdateServiceAccountRequest maps the mutable subset (PATCH
// /service-accounts/{id}). Both fields are optional; omitted fields are left
// unchanged by the API.
type UpdateServiceAccountRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

func (c *Client) CreateServiceAccount(ctx context.Context, req CreateServiceAccountRequest) (*ServiceAccount, error) {
	var out ServiceAccount
	if err := c.do(ctx, http.MethodPost, "/service-accounts", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetServiceAccount(ctx context.Context, id string) (*ServiceAccount, error) {
	var out ServiceAccount
	if err := c.do(ctx, http.MethodGet, "/service-accounts/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateServiceAccount(ctx context.Context, id string, req UpdateServiceAccountRequest) (*ServiceAccount, error) {
	var out ServiceAccount
	if err := c.do(ctx, http.MethodPatch, "/service-accounts/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteServiceAccount(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/service-accounts/"+url.PathEscape(id), nil, nil)
}
