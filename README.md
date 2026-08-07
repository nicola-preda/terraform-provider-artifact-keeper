# terraform-provider-artifact-keeper

Manage **Artifact Keeper** as code, through its REST API. The goal is the whole
configurable surface: every whole-object endpoint the API exposes, and every setting you
can change in the UI, is a Terraform resource.

Attribute names match the API's JSON fields one-to-one (`repo_type`, `is_public`, …), so
there's nothing to translate in your head.

Requires Terraform 1.5.7+ or OpenTofu. Tracks Artifact Keeper **1.7.1** (see
[MAINTAINING.md](MAINTAINING.md) for the coverage map and per-release upgrade checks).

## What you can manage

49 resources and 4 data sources. Each row is a job you'd otherwise do by clicking through
the admin UI; [`docs/`](docs/) has the full schema for every resource.

| What you want to do | Resources |
|---|---|
| Create hosted, proxy and virtual repositories, and set quotas, anonymous access, versioning, project ownership | `repository` |
| Tune a repository's proxy behaviour: cache TTL, upstream credentials, path rewrites, npm scope limits, PyPI tracks | `repository_cache_ttl`, `repository_upstream_auth`, `repository_routing_rules`, `repository_npm_scope_policy`, `repository_pypi_track` |
| Gate what enters a repository: vulnerability scanning, age-based holds on fresh upstream releases, curation rules | `repository_security`, `age_gate`, `curation_rule` |
| Expire and clean up artifacts on a schedule | `lifecycle_policy` |
| Promote artifacts between staging and release repositories, with rules and quality bars | `promotion_rule`, `repository_release_target`, `quality_gate`, `security_policy`, `license_policy` |
| Sign metadata and artifacts, and manage the signing keys | `signing_key`, `repository_signing_config` |
| Onboard teams: projects, groups, users, per-repository permissions | `project`, `project_membership`, `group`, `group_membership`, `user`, `user_role_assignment`, `permission` |
| Issue credentials for CI: service accounts, scoped tokens, per-repo tokens, keyless OIDC from your CI provider | `service_account`, `service_account_token`, `api_token`, `user_api_token`, `repo_token`, `ci_oidc_provider`, `ci_oidc_identity_mapping` |
| Wire up SSO against your IdP | `sso_oidc`, `sso_saml`, `sso_ldap` |
| Replicate between instances and target peers by label | `peer`, `peer_repository_subscription`, `peer_instance_label`, `peer_network_profile`, `sync_policy`, `remote_instance` |
| Import from a legacy registry (Nexus, Artifactory) | `migration_source`, `migration_job` |
| Notify on events | `webhook`, `email_subscription` |
| Set instance-wide behaviour: retention, upload limits, anonymous downloads, telemetry, which package formats are enabled, plugins | `system_settings`, `telemetry_settings`, `format_handler`, `plugin` |
| Look up objects you don't manage here, by their natural key | `data.artifactkeeper_repository`, `data.artifactkeeper_project`, `data.artifactkeeper_user`, `data.artifactkeeper_group` |

Deliberately **not** modelled, because they aren't declarative state: imperative actions
(approvals, promotion and migration runs, quarantine release, plugin installs, cache
invalidation, storage GC, backup runs), read-only and monitoring endpoints, the package
wire protocols themselves, and SMTP (env-configured upstream, with only a test endpoint).
[MAINTAINING.md](MAINTAINING.md#capability-gaps-backend-offers-provider-doesnt-model) has
the exact boundary.

## Usage

```hcl
terraform {
  required_providers {
    artifactkeeper = {
      source  = "nicola-preda/artifact-keeper"
      version = "~> 1.7.1"
    }
  }
}

provider "artifactkeeper" {
  endpoint = "https://artifact-keeper.example.com" # or ARTIFACT_KEEPER_ENDPOINT
  token    = var.token                             # or ARTIFACT_KEEPER_TOKEN
}

# A pull-through cache of the public npm registry.
resource "artifactkeeper_repository" "npm_proxy" {
  key          = "npm-remote"
  name         = "npm (proxy)"
  format       = "npm"
  repo_type    = "remote"
  upstream_url = "https://registry.npmjs.org"
  is_public    = true
}
```

**Auth:** a bearer `token` (an Artifact Keeper API token), or `username` + `password`
(traded for a token at `POST /api/v1/auth/login`). Any of these can come from the
environment: `ARTIFACT_KEEPER_TOKEN`, `ARTIFACT_KEEPER_USERNAME`, `ARTIFACT_KEEPER_PASSWORD`.

## Gotchas

- **Destroying a `repository`, `user`, or `group` is irreversible.** Deleting a repository
  takes every artifact with it; guard the important ones with
  `lifecycle { prevent_destroy = true }`.
- **Secrets are stored in state.** Credential fields are marked sensitive (masked in
  output) but still written to state; they aren't write-only (that needs Terraform
  1.11+/OpenTofu 1.11+, past the 1.5.7 floor). Use an encrypted backend, or OpenTofu's
  native state encryption.
- **Peers show `offline`** until the backend receives heartbeats. The upstream emitter
  isn't in place yet, so that status is expected.

## Dev and non-TLS instances

```hcl
# self-signed HTTPS: skip cert verification (insecure; dev / trusted networks only)
provider "artifactkeeper" {
  endpoint             = "https://artifact-keeper.dev.internal"
  insecure_skip_verify = true # or ARTIFACT_KEEPER_INSECURE_SKIP_VERIFY
}

# no TLS: a plain http:// endpoint (a bare host with no scheme defaults to https)
provider "artifactkeeper" {
  endpoint = "http://localhost:8080"
}
```

## Versioning

The provider version tracks the Artifact Keeper version it's validated against: `v1.7.1`
targets Artifact Keeper 1.7.1, and every release's acceptance suite is run against that
exact backend image. Pin with `~> 1.7.1`.

## Development

```sh
go build ./...
go test ./...   # unit tests; acceptance tests need a live instance (see MAINTAINING.md)
```

Point Terraform at a local build with `dev_overrides` in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides { "nicola-preda/artifact-keeper" = "/path/to/GOBIN" }
  direct {}
}
```
