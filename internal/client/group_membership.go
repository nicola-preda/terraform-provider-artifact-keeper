package client

import (
	"context"
	"fmt"
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

// ListGroupMembers returns all of a group's members via GET /groups/{id}
// (404 if gone), paging through with member_limit/member_offset until
// members_total is reached (the server caps member_limit at 200).
func (c *Client) ListGroupMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	const pageSize = 200
	var all []GroupMember
	for offset := 0; ; offset += pageSize {
		var out groupMembersEnvelope
		path := fmt.Sprintf("/groups/%s?member_limit=%d&member_offset=%d", url.PathEscape(groupID), pageSize, offset)
		if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
			return nil, err
		}
		all = append(all, out.Members...)
		// Stop once we have them all, or defensively if a page comes back empty
		// (guards against a stale members_total).
		if int64(len(all)) >= out.MembersTotal || len(out.Members) == 0 {
			break
		}
	}
	return all, nil
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
