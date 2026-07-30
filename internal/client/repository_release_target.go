package client

import (
	"context"
	"net/http"
	"net/url"
)

// ReleaseTargetResponse is the body of GET /promotion/repositories/{key}/release-target:
// whether the staging repo is linked to a release repo, and which one. The
// *string fields are nil when nothing is linked.
type ReleaseTargetResponse struct {
	Linked               bool    `json:"linked"`
	ReleaseRepositoryKey *string `json:"release_repository_key"`
	ReleaseRepositoryID  *string `json:"release_repository_id"`
}

// SetReleaseTargetRequest maps the PUT body. A nil/omitted ReleaseRepositoryKey
// removes the link.
type SetReleaseTargetRequest struct {
	ReleaseRepositoryKey *string `json:"release_repository_key"`
}

// GetReleaseTarget reads GET /promotion/repositories/{key}/release-target.
func (c *Client) GetReleaseTarget(ctx context.Context, repoKey string) (*ReleaseTargetResponse, error) {
	var out ReleaseTargetResponse
	if err := c.do(ctx, http.MethodGet, "/promotion/repositories/"+url.PathEscape(repoKey)+"/release-target", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetReleaseTarget upserts via PUT /promotion/repositories/{key}/release-target.
// Passing a nil release key unlinks the staging repo. Read the result back with
// GetReleaseTarget.
func (c *Client) SetReleaseTarget(ctx context.Context, repoKey string, req SetReleaseTargetRequest) error {
	return c.do(ctx, http.MethodPut, "/promotion/repositories/"+url.PathEscape(repoKey)+"/release-target", req, nil)
}
