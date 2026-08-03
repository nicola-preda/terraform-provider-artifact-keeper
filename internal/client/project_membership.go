package client

import (
	"context"
	"net/http"
	"net/url"
)

// ProjectMember mirrors ProjectMemberRow (a permissions row with
// target_type = 'project'): a grant of `actions` on every repository in the
// project to a user or group.
type ProjectMember struct {
	PrincipalType string   `json:"principal_type"`
	PrincipalID   string   `json:"principal_id"`
	Actions       []string `json:"actions"`
}

type projectMembersListResponse struct {
	Items []ProjectMember `json:"items"`
}

// AddProjectMemberRequest maps AddProjectMemberRequest. POST upserts: re-adding
// an existing principal replaces its action set.
type AddProjectMemberRequest struct {
	PrincipalType string   `json:"principal_type"`
	PrincipalID   string   `json:"principal_id"`
	Actions       []string `json:"actions"`
}

type removeProjectMemberRequest struct {
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
}

// SetProjectMember upserts a grant via POST /projects/{id}/members and returns
// the stored row.
func (c *Client) SetProjectMember(ctx context.Context, projectID string, req AddProjectMemberRequest) (*ProjectMember, error) {
	var out ProjectMember
	if err := c.do(ctx, http.MethodPost, "/projects/"+url.PathEscape(projectID)+"/members", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetProjectMember lists a project's members and returns the grant for
// (principalType, principalID); 404 when that principal has no grant. There is
// no single-member GET, so list and filter (mirrors GetGroupMember).
func (c *Client) GetProjectMember(ctx context.Context, projectID, principalType, principalID string) (*ProjectMember, error) {
	var out projectMembersListResponse
	if err := c.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/members", nil, &out); err != nil {
		return nil, err
	}
	for i := range out.Items {
		if out.Items[i].PrincipalType == principalType && out.Items[i].PrincipalID == principalID {
			return &out.Items[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: "project membership not found"}
}

// RemoveProjectMember deletes a grant via DELETE /projects/{id}/members (the
// principal is identified in the request body). Removing an absent grant is a
// no-op.
func (c *Client) RemoveProjectMember(ctx context.Context, projectID, principalType, principalID string) error {
	body := removeProjectMemberRequest{PrincipalType: principalType, PrincipalID: principalID}
	return c.do(ctx, http.MethodDelete, "/projects/"+url.PathEscape(projectID)+"/members", body, nil)
}
