package client

import (
	"context"
	"net/http"
	"net/url"
)

// User mirrors AdminUserResponse. Password material is never returned.
type User struct {
	ID                 string  `json:"id"`
	Username           string  `json:"username"`
	Email              string  `json:"email"`
	DisplayName        *string `json:"display_name"`
	AuthProvider       string  `json:"auth_provider"`
	IsActive           bool    `json:"is_active"`
	IsAdmin            bool    `json:"is_admin"`
	MustChangePassword bool    `json:"must_change_password"`
	LastLoginAt        *string `json:"last_login_at"`
	CreatedAt          string  `json:"created_at"`
}

type CreateUserRequest struct {
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	Password    *string `json:"password,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	IsAdmin     *bool   `json:"is_admin,omitempty"`
}

// CreateUserResponse wraps the user plus the auto-generated password (returned
// only when no password was supplied).
type CreateUserResponse struct {
	User              User    `json:"user"`
	GeneratedPassword *string `json:"generated_password"`
}

type UpdateUserRequest struct {
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
	IsAdmin     *bool   `json:"is_admin,omitempty"`
}

func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
	var out CreateUserResponse
	if err := c.do(ctx, http.MethodPost, "/users", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var out User
	if err := c.do(ctx, http.MethodGet, "/users/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateUser(ctx context.Context, id string, req UpdateUserRequest) (*User, error) {
	var out User
	if err := c.do(ctx, http.MethodPatch, "/users/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteUser(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/users/"+url.PathEscape(id), nil, nil)
}
