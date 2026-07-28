package client

import (
	"context"
	"net/http"
	"net/url"
)

// Peer mirrors PeerInstanceResponse. The api_key is never returned.
type Peer struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	EndpointURL     string  `json:"endpoint_url"`
	Status          string  `json:"status"`
	Region          *string `json:"region"`
	CacheSizeBytes  int64   `json:"cache_size_bytes"`
	CacheUsedBytes  int64   `json:"cache_used_bytes"`
	LastHeartbeatAt *string `json:"last_heartbeat_at"`
	LastSyncAt      *string `json:"last_sync_at"`
	CreatedAt       string  `json:"created_at"`
	IsLocal         bool    `json:"is_local"`
}

// RegisterPeerRequest maps RegisterPeerRequest. sync_filter is omitted for now.
type RegisterPeerRequest struct {
	Name           string  `json:"name"`
	EndpointURL    string  `json:"endpoint_url"`
	Region         *string `json:"region,omitempty"`
	CacheSizeBytes *int64  `json:"cache_size_bytes,omitempty"`
	APIKey         string  `json:"api_key"`
}

func (c *Client) CreatePeer(ctx context.Context, req RegisterPeerRequest) (*Peer, error) {
	var out Peer
	if err := c.do(ctx, http.MethodPost, "/peers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPeer(ctx context.Context, id string) (*Peer, error) {
	var out Peer
	if err := c.do(ctx, http.MethodGet, "/peers/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeletePeer(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/peers/"+url.PathEscape(id), nil, nil)
}
