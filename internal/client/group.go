package client

import (
	"context"
	"net/http"
	"net/url"
)

// Group mirrors GroupResponse (extra fields like members on GET are ignored).
type Group struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	MemberCount    int64   `json:"member_count"`
	ExternalSource *string `json:"external_source"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// GroupRequest is used for both create (POST) and update (PUT).
type GroupRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

func (c *Client) CreateGroup(ctx context.Context, req GroupRequest) (*Group, error) {
	var out Group
	if err := c.do(ctx, http.MethodPost, "/groups", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetGroup(ctx context.Context, id string) (*Group, error) {
	var out Group
	if err := c.do(ctx, http.MethodGet, "/groups/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateGroup(ctx context.Context, id string, req GroupRequest) (*Group, error) {
	var out Group
	if err := c.do(ctx, http.MethodPut, "/groups/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteGroup(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/groups/"+url.PathEscape(id), nil, nil)
}
