package client

import (
	"context"
	"net/http"
	"net/url"
)

// PeerInstanceLabel mirrors PeerLabelResponse: a single key/value label on a
// peer instance, addressed by (peer id, label key). Sync policies match on
// these labels, and the backend re-evaluates them on every label write.
type PeerInstanceLabel struct {
	ID             string `json:"id"`
	PeerInstanceID string `json:"peer_instance_id"`
	Key            string `json:"key"`
	Value          string `json:"value"`
	CreatedAt      string `json:"created_at"`
}

// peerInstanceLabelsListResponse mirrors PeerLabelsListResponse.
type peerInstanceLabelsListResponse struct {
	Items []PeerInstanceLabel `json:"items"`
	Total int64               `json:"total"`
}

// setPeerInstanceLabelRequest maps AddPeerLabelRequest.
type setPeerInstanceLabelRequest struct {
	Value string `json:"value"`
}

// ListPeerInstanceLabels returns every label set on a peer instance. A missing
// peer surfaces as a 404 APIError.
func (c *Client) ListPeerInstanceLabels(ctx context.Context, peerID string) ([]PeerInstanceLabel, error) {
	var out peerInstanceLabelsListResponse
	if err := c.do(ctx, http.MethodGet, "/peers/"+url.PathEscape(peerID)+"/labels", nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// GetPeerInstanceLabel finds a single label by key. The API has no single-label
// GET, so it lists and filters. Returns a 404 APIError when absent.
func (c *Client) GetPeerInstanceLabel(ctx context.Context, peerID, labelKey string) (*PeerInstanceLabel, error) {
	labels, err := c.ListPeerInstanceLabels(ctx, peerID)
	if err != nil {
		return nil, err
	}
	for i := range labels {
		if labels[i].Key == labelKey {
			return &labels[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: "peer instance label not found"}
}

// SetPeerInstanceLabel adds or updates (upserts) a single label via
// POST /peers/{id}/labels/{label_key}. The whole-set PUT is deliberately not
// used: one resource per label composes without fighting Terraform over
// ownership of the rest of the set.
func (c *Client) SetPeerInstanceLabel(ctx context.Context, peerID, labelKey, value string) (*PeerInstanceLabel, error) {
	var out PeerInstanceLabel
	body := setPeerInstanceLabelRequest{Value: value}
	if err := c.do(ctx, http.MethodPost, "/peers/"+url.PathEscape(peerID)+"/labels/"+url.PathEscape(labelKey), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeletePeerInstanceLabel(ctx context.Context, peerID, labelKey string) error {
	return c.do(ctx, http.MethodDelete, "/peers/"+url.PathEscape(peerID)+"/labels/"+url.PathEscape(labelKey), nil, nil)
}
