package client

import (
	"context"
	"net/http"
	"net/url"
)

// GroupMember mirrors GroupMemberResponse from GET /groups/{id}. Membership is
// the (group id, user id) edge; the other fields are server-returned only.
type GroupMember struct {
	UserID      string  `json:"user_id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name"`
	JoinedAt    string  `json:"joined_at"`
}

// groupMembersEnvelope decodes only the members of GroupDetailResponse
// (GET /groups/{id}); group fields are ignored.
type groupMembersEnvelope struct {
	Members      []GroupMember `json:"members"`
	MembersTotal int64         `json:"members_total"`
}

// membersRequest maps MembersRequest: a batch of user ids to add or remove. We
// always send exactly one (one membership per resource).
type membersRequest struct {
	UserIDs []string `json:"user_ids"`
}

// AddGroupMember adds userID to groupID via POST /groups/{id}/members. Upserts
// (ON CONFLICT DO NOTHING), so re-adding an existing member is a no-op.
func (c *Client) AddGroupMember(ctx context.Context, groupID, userID string) error {
	body := membersRequest{UserIDs: []string{userID}}
	return c.do(ctx, http.MethodPost, "/groups/"+url.PathEscape(groupID)+"/members", body, nil)
}

// RemoveGroupMember removes userID from groupID via DELETE /groups/{id}/members.
// Removing an absent member is a no-op.
func (c *Client) RemoveGroupMember(ctx context.Context, groupID, userID string) error {
	body := membersRequest{UserIDs: []string{userID}}
	return c.do(ctx, http.MethodDelete, "/groups/"+url.PathEscape(groupID)+"/members", body, nil)
}

// ListGroupMembers returns a group's members via GET /groups/{id} (404 if gone).
// Only the first page is read (server default 50); pagination isn't wired up, so
// members past 50 won't surface.
func (c *Client) ListGroupMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	var out groupMembersEnvelope
	if err := c.do(ctx, http.MethodGet, "/groups/"+url.PathEscape(groupID), nil, &out); err != nil {
		return nil, err
	}
	return out.Members, nil
}

// GetGroupMember confirms the (groupID, userID) edge. No single-membership GET,
// so list and filter; 404 when the edge is absent (removed member or deleted
// group).
func (c *Client) GetGroupMember(ctx context.Context, groupID, userID string) (*GroupMember, error) {
	members, err := c.ListGroupMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	for i := range members {
		if members[i].UserID == userID {
			return &members[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: "group membership not found"}
}
