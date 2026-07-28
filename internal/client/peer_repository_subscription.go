package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// PeerRepositorySubscription mirrors SubscriptionResponse (GET
// /peers/{id}/repositories/{repo_id}): replication config for one (peer,
// repository) pair.
type PeerRepositorySubscription struct {
	ID             string `json:"id"`
	PeerInstanceID string `json:"peer_instance_id"`
	RepositoryID   string `json:"repository_id"`
	SyncEnabled    bool   `json:"sync_enabled"`
	// ReplicationMode is push/pull/mirror/none. Nullable; backend defaults an
	// omitted mode to "pull" on write.
	ReplicationMode     *string `json:"replication_mode"`
	ReplicationSchedule *string `json:"replication_schedule"`
	// ReplicationFilter is a free-form JSONB object limiting which artifacts
	// replicate. JSON null when unset; passed through verbatim.
	ReplicationFilter json.RawMessage `json:"replication_filter"`
	LastReplicatedAt  *string         `json:"last_replicated_at"`
	CreatedAt         string          `json:"created_at"`
}

// AssignRepoRequest maps AssignRepoRequest (POST /peers/{id}/repositories). Only
// repository_id required (sync_enabled defaults true, replication_mode "pull").
// Upserts on (peer, repository), so re-POSTing updates in place.
type AssignRepoRequest struct {
	RepositoryID        string          `json:"repository_id"`
	SyncEnabled         *bool           `json:"sync_enabled,omitempty"`
	ReplicationMode     *string         `json:"replication_mode,omitempty"`
	ReplicationSchedule *string         `json:"replication_schedule,omitempty"`
	ReplicationFilter   json.RawMessage `json:"replication_filter,omitempty"`
}

// AssignRepo subscribes a repo to peerID, or updates the existing subscription
// (upsert on (peer, repository)). Returns no body; read it back with
// GetPeerRepositorySubscription.
func (c *Client) AssignRepo(ctx context.Context, peerID string, req AssignRepoRequest) error {
	return c.do(ctx, http.MethodPost, "/peers/"+url.PathEscape(peerID)+"/repositories", req, nil)
}

// GetPeerRepositorySubscription fetches one (peer, repository) subscription; 404
// when none exists.
func (c *Client) GetPeerRepositorySubscription(ctx context.Context, peerID, repoID string) (*PeerRepositorySubscription, error) {
	var out PeerRepositorySubscription
	if err := c.do(ctx, http.MethodGet, "/peers/"+url.PathEscape(peerID)+"/repositories/"+url.PathEscape(repoID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UnassignRepo removes the subscription for a (peer, repository) pair.
func (c *Client) UnassignRepo(ctx context.Context, peerID, repoID string) error {
	return c.do(ctx, http.MethodDelete, "/peers/"+url.PathEscape(peerID)+"/repositories/"+url.PathEscape(repoID), nil, nil)
}
