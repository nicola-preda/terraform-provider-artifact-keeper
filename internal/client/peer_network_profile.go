package client

import (
	"context"
	"net/http"
	"net/url"
)

// NetworkProfile mirrors NetworkProfileBody (PUT /peers/{id}/network-profile):
// transfer scheduling and bandwidth for one peer. Every field optional.
// Write-only: no GET on the profile, and the peer GET doesn't return these.
type NetworkProfile struct {
	// MaxBandwidthBps caps outbound bandwidth to the peer, in bytes/sec.
	MaxBandwidthBps *int64 `json:"max_bandwidth_bps,omitempty"`
	// SyncWindowStart/End bound the daily sync window, as "HH:MM:SS" 24-hour times.
	SyncWindowStart *string `json:"sync_window_start,omitempty"`
	SyncWindowEnd   *string `json:"sync_window_end,omitempty"`
	// SyncWindowTimezone is the IANA timezone the window is read in (e.g. "UTC").
	SyncWindowTimezone *string `json:"sync_window_timezone,omitempty"`
	// ConcurrentTransfersLimit caps in-flight transfers to the peer.
	ConcurrentTransfersLimit *int32 `json:"concurrent_transfers_limit,omitempty"`
}

// UpdatePeerNetworkProfile PUTs the profile for peerID. Returns no body; no
// matching read endpoint. A nil field is omitted; the backend COALESCEs it to
// the stored value rather than clearing it.
func (c *Client) UpdatePeerNetworkProfile(ctx context.Context, peerID string, req NetworkProfile) error {
	return c.do(ctx, http.MethodPut, "/peers/"+url.PathEscape(peerID)+"/network-profile", req, nil)
}
