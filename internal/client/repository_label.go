package client

import (
	"context"
	"net/http"
	"net/url"
)

// RepositoryLabel mirrors LabelResponse: a single key/value label on a
// repository, addressed by (repository key, label key).
type RepositoryLabel struct {
	ID           string `json:"id"`
	RepositoryID string `json:"repository_id"`
	Key          string `json:"key"`
	Value        string `json:"value"`
	CreatedAt    string `json:"created_at"`
}

// repositoryLabelsListResponse mirrors LabelsListResponse.
type repositoryLabelsListResponse struct {
	Items []RepositoryLabel `json:"items"`
	Total int64             `json:"total"`
}

// setRepositoryLabelRequest maps AddLabelRequest (value defaults to "" server
// side when omitted).
type setRepositoryLabelRequest struct {
	Value string `json:"value"`
}

// ListRepositoryLabels returns every label set on a repository. A missing
// repository surfaces as a 404 APIError.
func (c *Client) ListRepositoryLabels(ctx context.Context, repoKey string) ([]RepositoryLabel, error) {
	var out repositoryLabelsListResponse
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repoKey)+"/labels", nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// GetRepositoryLabel finds a single label by key. The API has no single-label
// GET, so it lists and filters. Returns a 404 APIError when absent.
func (c *Client) GetRepositoryLabel(ctx context.Context, repoKey, labelKey string) (*RepositoryLabel, error) {
	labels, err := c.ListRepositoryLabels(ctx, repoKey)
	if err != nil {
		return nil, err
	}
	for i := range labels {
		if labels[i].Key == labelKey {
			return &labels[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: "repository label not found"}
}

// SetRepositoryLabel adds or updates (upserts) a single label via
// POST /repositories/{key}/labels/{label_key}.
func (c *Client) SetRepositoryLabel(ctx context.Context, repoKey, labelKey, value string) (*RepositoryLabel, error) {
	var out RepositoryLabel
	body := setRepositoryLabelRequest{Value: value}
	if err := c.do(ctx, http.MethodPost, "/repositories/"+url.PathEscape(repoKey)+"/labels/"+url.PathEscape(labelKey), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRepositoryLabel(ctx context.Context, repoKey, labelKey string) error {
	return c.do(ctx, http.MethodDelete, "/repositories/"+url.PathEscape(repoKey)+"/labels/"+url.PathEscape(labelKey), nil, nil)
}
