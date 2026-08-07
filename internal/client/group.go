package client

import (
	"context"
	"fmt"
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

// FindGroupByName resolves a group name to its group. The search endpoint does
// a substring match, so we filter the results for an exact name.
func (c *Client) FindGroupByName(ctx context.Context, name string) (*Group, error) {
	q := url.Values{}
	q.Set("search", name)
	q.Set("per_page", "100") // ponytail: an exact match hiding past 100 substring collisions is pathological
	var out struct {
		Items []Group `json:"items"`
	}
	if err := c.do(ctx, http.MethodGet, "/groups?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	for i := range out.Items {
		if out.Items[i].Name == name {
			return &out.Items[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("no group named %q", name)}
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
