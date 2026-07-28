package client

import (
	"context"
	"net/http"
	"net/url"
)

// Project mirrors the API Project model (GET /projects/{id}): a metadata
// grouping of repositories.
type Project struct {
	ID          string  `json:"id"`
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	QuotaBytes  *int64  `json:"quota_bytes"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// CreateProjectRequest maps POST /projects. key and name required; key must be
// unique and URL-safe.
type CreateProjectRequest struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	QuotaBytes  *int64  `json:"quota_bytes,omitempty"`
}

// UpdateProjectRequest maps PUT /projects/{id}. Omitted (nil) fields are left
// unchanged (COALESCE); key is immutable.
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	QuotaBytes  *int64  `json:"quota_bytes,omitempty"`
}

func (c *Client) CreateProject(ctx context.Context, req CreateProjectRequest) (*Project, error) {
	var out Project
	if err := c.do(ctx, http.MethodPost, "/projects", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	var out Project
	if err := c.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateProject(ctx context.Context, id string, req UpdateProjectRequest) (*Project, error) {
	var out Project
	if err := c.do(ctx, http.MethodPut, "/projects/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/projects/"+url.PathEscape(id), nil, nil)
}
