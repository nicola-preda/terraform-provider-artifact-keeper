package client

import (
	"context"
	"net/http"
	"net/url"
)

// UpstreamAuthRequest maps the PUT /repositories/{key}/upstream-auth body: the
// credentials a remote repository uses to authenticate to its upstream.
// auth_type is one of "basic", "bearer", "aws_ecr", "aws_codeartifact" or
// "none" ("none" removes the auth). password carries the basic password or the
// bearer token; it is never returned.
type UpstreamAuthRequest struct {
	AuthType string  `json:"auth_type"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	// Aws is required for the two dynamic AWS auth types (1.10.0, #1559) and
	// ignored otherwise. It carries no secret: the AWS identity comes from the
	// backend process's own credential chain.
	Aws *AwsUpstreamAuth `json:"aws,omitempty"`
}

// AwsUpstreamAuth maps AwsUpstreamAuthRequest, the non-secret provider settings
// of an aws_ecr or aws_codeartifact upstream.
type AwsUpstreamAuth struct {
	// Region of the registry or domain, e.g. us-east-1.
	Region string `json:"region"`
	// RegistryID pins the ECR upstream host to an account id. ECR only.
	RegistryID *string `json:"registry_id,omitempty"`
	// Domain is the CodeArtifact domain, and is required for that auth type.
	Domain *string `json:"domain,omitempty"`
	// DomainOwner defaults to the caller's account. CodeArtifact only.
	DomainOwner *string `json:"domain_owner,omitempty"`
	// DurationSeconds is the requested token lifetime (0, or 900..=43200),
	// defaulting to AWS's own 12 hours. CodeArtifact only.
	DurationSeconds *int64 `json:"duration_seconds,omitempty"`
}

// SetUpstreamAuth writes the upstream credentials via PUT. Write-only: there is
// no GET, and the response carries nothing we keep, so it is discarded.
func (c *Client) SetUpstreamAuth(ctx context.Context, repoKey string, req UpstreamAuthRequest) error {
	return c.do(ctx, http.MethodPut, "/repositories/"+url.PathEscape(repoKey)+"/upstream-auth", req, nil)
}
