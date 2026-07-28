package client

import (
	"context"
	"net/http"
	"net/url"
)

// EmailSubscription mirrors EmailSubscriptionResponse, keyed by (repository key,
// id). No single-item GET, so Read lists and filters by id.
type EmailSubscription struct {
	ID           string   `json:"id"`
	RepositoryID *string  `json:"repository_id"`
	Recipients   []string `json:"recipients"`
	EventTypes   []string `json:"event_types"`
	Enabled      bool     `json:"enabled"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// CreateEmailSubscriptionRequest: recipients and event_types required; enabled
// defaults to true server-side.
type CreateEmailSubscriptionRequest struct {
	Recipients []string `json:"recipients"`
	EventTypes []string `json:"event_types"`
	Enabled    *bool    `json:"enabled,omitempty"`
}

// emailSubscriptionListResponse mirrors EmailSubscriptionListResponse.
type emailSubscriptionListResponse struct {
	Subscriptions []EmailSubscription `json:"subscriptions"`
}

// CreateEmailSubscription posts to /repositories/{key}/email-subscriptions.
func (c *Client) CreateEmailSubscription(ctx context.Context, repoKey string, req CreateEmailSubscriptionRequest) (*EmailSubscription, error) {
	var out EmailSubscription
	if err := c.do(ctx, http.MethodPost, "/repositories/"+url.PathEscape(repoKey)+"/email-subscriptions", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEmailSubscriptions lists a repository's subscriptions; a missing
// repository is a 404.
func (c *Client) ListEmailSubscriptions(ctx context.Context, repoKey string) ([]EmailSubscription, error) {
	var out emailSubscriptionListResponse
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repoKey)+"/email-subscriptions", nil, &out); err != nil {
		return nil, err
	}
	return out.Subscriptions, nil
}

// DeleteEmailSubscription deletes by id:
// DELETE /repositories/{key}/email-subscriptions/{subscription_id}.
func (c *Client) DeleteEmailSubscription(ctx context.Context, repoKey, id string) error {
	return c.do(ctx, http.MethodDelete, "/repositories/"+url.PathEscape(repoKey)+"/email-subscriptions/"+url.PathEscape(id), nil, nil)
}
