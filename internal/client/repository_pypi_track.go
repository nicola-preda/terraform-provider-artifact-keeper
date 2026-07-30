package client

import (
	"context"
	"net/http"
	"net/url"
)

// PypiTrackRequest is the body of PUT /repositories/{key}/pypi-tracks/{project}:
// the upstream Simple index URL the local project mirrors.
type PypiTrackRequest struct {
	TracksURL string `json:"tracks_url"`
}

// PypiTrackResponse mirrors one item of GET /repositories/{key}/pypi-tracks. The
// backend PEP 503-normalizes the {project} path into normalized_name.
type PypiTrackResponse struct {
	RepositoryKey  string `json:"repository_key"`
	NormalizedName string `json:"normalized_name"`
	TracksURL      string `json:"tracks_url"`
}

// PypiTracksListResponse wraps the list body; only items is read.
type PypiTracksListResponse struct {
	Items []PypiTrackResponse `json:"items"`
}

// SetPypiTrack upserts the track for {project} on {key} via PUT. Admin only.
// Read it back with GetPypiTrack (the project is normalized server-side).
func (c *Client) SetPypiTrack(ctx context.Context, repoKey, project string, req PypiTrackRequest) error {
	return c.do(ctx, http.MethodPut, "/repositories/"+url.PathEscape(repoKey)+"/pypi-tracks/"+url.PathEscape(project), req, nil)
}

// DeletePypiTrack removes the track for {project} on {key} via DELETE.
func (c *Client) DeletePypiTrack(ctx context.Context, repoKey, project string) error {
	return c.do(ctx, http.MethodDelete, "/repositories/"+url.PathEscape(repoKey)+"/pypi-tracks/"+url.PathEscape(project), nil, nil)
}

// ListPypiTracks returns every track configured on {key} via GET.
func (c *Client) ListPypiTracks(ctx context.Context, repoKey string) ([]PypiTrackResponse, error) {
	var out PypiTracksListResponse
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repoKey)+"/pypi-tracks", nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// GetPypiTrack finds a single track by its normalized name. There is no
// single-track GET, so list and filter; 404 when the track is absent (removed
// track or deleted repository).
func (c *Client) GetPypiTrack(ctx context.Context, repoKey, normalizedName string) (*PypiTrackResponse, error) {
	items, err := c.ListPypiTracks(ctx, repoKey)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].NormalizedName == normalizedName {
			return &items[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: "pypi track not found"}
}
