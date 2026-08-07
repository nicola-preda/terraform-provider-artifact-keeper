package client

import (
	"context"
	"fmt"
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

// FindUserByUsername resolves a username to its user. The search endpoint does
// a substring match, so we filter the results for an exact username.
func (c *Client) FindUserByUsername(ctx context.Context, username string) (*User, error) {
	q := url.Values{}
	q.Set("search", username)
	q.Set("per_page", "100") // ponytail: an exact match hiding past 100 substring collisions is pathological
	var out struct {
		Items []User `json:"items"`
	}
	if err := c.do(ctx, http.MethodGet, "/users?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	for i := range out.Items {
		if out.Items[i].Username == username {
			return &out.Items[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("no user with username %q", username)}
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
