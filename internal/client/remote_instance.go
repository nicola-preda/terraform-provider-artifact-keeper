package client

import (
	"context"
	"net/http"
	"net/url"
)

// RemoteInstance mirrors RemoteInstanceResponse (a registered remote Artifact
// Keeper instance the API can proxy to). The API key is never returned.
type RemoteInstance struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

// CreateRemoteInstanceRequest maps CreateInstanceRequest. All fields are
// required by the API.
type CreateRemoteInstanceRequest struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// ListRemoteInstances returns the remote instances owned by the authenticated
// user. The API exposes no single-instance GET, so callers read by listing.
func (c *Client) ListRemoteInstances(ctx context.Context) ([]RemoteInstance, error) {
	var out []RemoteInstance
	if err := c.do(ctx, http.MethodGet, "/instances", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRemoteInstance finds a single instance by id by listing (no GET /:id).
// Returns a 404 APIError when no instance matches.
func (c *Client) GetRemoteInstance(ctx context.Context, id string) (*RemoteInstance, error) {
	instances, err := c.ListRemoteInstances(ctx)
	if err != nil {
		return nil, err
	}
	for i := range instances {
		if instances[i].ID == id {
			return &instances[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: "remote instance not found"}
}

func (c *Client) CreateRemoteInstance(ctx context.Context, req CreateRemoteInstanceRequest) (*RemoteInstance, error) {
	var out RemoteInstance
	if err := c.do(ctx, http.MethodPost, "/instances", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRemoteInstance(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/instances/"+url.PathEscape(id), nil, nil)
}
