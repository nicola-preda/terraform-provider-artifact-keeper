package client

import (
	"context"
	"net/http"
	"net/url"
)

// Permission mirrors PermissionResponse: a grant of actions to a principal
// (user/group) on a target (repository/group/system/...).
type Permission struct {
	ID            string   `json:"id"`
	PrincipalType string   `json:"principal_type"`
	PrincipalID   string   `json:"principal_id"`
	PrincipalName *string  `json:"principal_name"`
	TargetType    string   `json:"target_type"`
	TargetID      string   `json:"target_id"`
	TargetName    *string  `json:"target_name"`
	Actions       []string `json:"actions"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// PermissionRequest is used for both create (POST) and update (PUT).
type PermissionRequest struct {
	PrincipalType string   `json:"principal_type"`
	PrincipalID   string   `json:"principal_id"`
	TargetType    string   `json:"target_type"`
	TargetID      string   `json:"target_id"`
	Actions       []string `json:"actions"`
}

func (c *Client) CreatePermission(ctx context.Context, req PermissionRequest) (*Permission, error) {
	var out Permission
	if err := c.do(ctx, http.MethodPost, "/permissions", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPermission(ctx context.Context, id string) (*Permission, error) {
	var out Permission
	if err := c.do(ctx, http.MethodGet, "/permissions/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdatePermission(ctx context.Context, id string, req PermissionRequest) (*Permission, error) {
	var out Permission
	if err := c.do(ctx, http.MethodPut, "/permissions/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeletePermission(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/permissions/"+url.PathEscape(id), nil, nil)
}
