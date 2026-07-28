package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// UserRole mirrors RoleResponse from GET /users/{id}/roles. The assignment
// carries nothing beyond the role; not repository-scoped.
type UserRole struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

type userRoleListResponse struct {
	Items []UserRole `json:"items"`
}

// AssignUserRoleRequest maps AssignRoleRequest (POST /users/{id}/roles). role_id
// only; not repository-scoped.
type AssignUserRoleRequest struct {
	RoleID string `json:"role_id"`
}

// AssignUserRole grants roleID to userID. POST is idempotent server-side
// (ON CONFLICT DO NOTHING).
func (c *Client) AssignUserRole(ctx context.Context, userID, roleID string) error {
	return c.do(ctx, http.MethodPost, "/users/"+url.PathEscape(userID)+"/roles", AssignUserRoleRequest{RoleID: roleID}, nil)
}

// GetUserRole confirms userID still holds roleID. No GET-by-id for a single
// assignment, so list and filter; returns 404 when the assignment is gone.
func (c *Client) GetUserRole(ctx context.Context, userID, roleID string) (*UserRole, error) {
	var list userRoleListResponse
	if err := c.do(ctx, http.MethodGet, "/users/"+url.PathEscape(userID)+"/roles", nil, &list); err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].ID == roleID {
			return &list.Items[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("role %s is not assigned to user %s", roleID, userID)}
}

// RemoveUserRole revokes roleID from userID.
func (c *Client) RemoveUserRole(ctx context.Context, userID, roleID string) error {
	return c.do(ctx, http.MethodDelete, "/users/"+url.PathEscape(userID)+"/roles/"+url.PathEscape(roleID), nil, nil)
}
