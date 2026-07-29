package client

import (
	"context"
	"net/http"
	"net/url"
)

// MigrationConfig is the config object on a migration job. Absent fields fall
// back to the backend defaults (include_cached_remote false, conflict_resolution
// "skip", concurrent_transfers 4, throttle_delay_ms 100, verify_checksums true,
// include_users/groups/permissions true).
type MigrationConfig struct {
	IncludeRepos        []string `json:"include_repos,omitempty"`
	ExcludeRepos        []string `json:"exclude_repos,omitempty"`
	ExcludePaths        []string `json:"exclude_paths,omitempty"`
	IncludeUsers        *bool    `json:"include_users,omitempty"`
	IncludeGroups       *bool    `json:"include_groups,omitempty"`
	IncludePermissions  *bool    `json:"include_permissions,omitempty"`
	IncludeCachedRemote *bool    `json:"include_cached_remote,omitempty"`
	DryRun              *bool    `json:"dry_run,omitempty"`
	ConflictResolution  *string  `json:"conflict_resolution,omitempty"`
	ConcurrentTransfers *int64   `json:"concurrent_transfers,omitempty"`
	ThrottleDelayMs     *int64   `json:"throttle_delay_ms,omitempty"`
	VerifyChecksums     *bool    `json:"verify_checksums,omitempty"`
	DateFrom            *string  `json:"date_from,omitempty"`
	DateTo              *string  `json:"date_to,omitempty"`
}

// CreateMigrationRequest maps the POST /migrations body.
type CreateMigrationRequest struct {
	SourceConnectionID string          `json:"source_connection_id"`
	JobType            *string         `json:"job_type,omitempty"`
	Config             MigrationConfig `json:"config"`
}

// MigrationJob mirrors MigrationJobResponse. A created job is pending; starting
// it is a separate operation this client does not perform.
type MigrationJob struct {
	ID                     string  `json:"id"`
	SourceConnectionID     string  `json:"source_connection_id"`
	Status                 string  `json:"status"`
	JobType                string  `json:"job_type"`
	TotalItems             int64   `json:"total_items"`
	CompletedItems         int64   `json:"completed_items"`
	FailedItems            int64   `json:"failed_items"`
	SkippedItems           int64   `json:"skipped_items"`
	TotalBytes             int64   `json:"total_bytes"`
	TransferredBytes       int64   `json:"transferred_bytes"`
	ProgressPercent        float64 `json:"progress_percent"`
	EstimatedTimeRemaining *int64  `json:"estimated_time_remaining"`
	CreatedAt              string  `json:"created_at"`
	StartedAt              *string `json:"started_at"`
	FinishedAt             *string `json:"finished_at"`
	ErrorSummary           *string `json:"error_summary"`
}

func (c *Client) CreateMigrationJob(ctx context.Context, req CreateMigrationRequest) (*MigrationJob, error) {
	var out MigrationJob
	if err := c.do(ctx, http.MethodPost, "/migrations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetMigrationJob(ctx context.Context, id string) (*MigrationJob, error) {
	var out MigrationJob
	if err := c.do(ctx, http.MethodGet, "/migrations/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteMigrationJob(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/migrations/"+url.PathEscape(id), nil, nil)
}
