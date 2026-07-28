# terraform-provider-artifact-keeper

Manage **Artifact Keeper** as code: repositories, users, SSO, peering, signing, promotion,
and policies, all driven through its REST API.

Attribute names match the API's JSON fields one-to-one (`repo_type`, `is_public`, …), so
there's nothing to translate in your head.

Requires Terraform 1.5.7+ or OpenTofu. Tracks Artifact Keeper **1.6.3** (see
[MAINTAINING.md](MAINTAINING.md) for the coverage map and per-release upgrade checks).

## Usage

```hcl
terraform {
  required_providers {
    artifactkeeper = {
      source  = "nicola-preda/artifact-keeper"
      version = "~> 1.6.3"
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

## Resources

37 resources. See [`docs/`](docs/) for each one's schema.

| Area | Resources |
|---|---|
| Repositories | `repository`, `repository_label`, `repository_signing_config`, `lifecycle_policy`, `age_gate` |
| Projects | `project` |
| Identity & access | `user`, `group`, `group_membership`, `user_role_assignment`, `permission`, `service_account`, `service_account_token`, `api_token`, `repo_token` |
| SSO & auth | `sso_oidc`, `sso_ldap`, `sso_saml`, `ci_oidc_provider`, `ci_oidc_identity_mapping` |
| Replication & migration | `peer`, `peer_repository_subscription`, `peer_network_profile`, `remote_instance`, `sync_policy`, `migration_source` |
| Promotion & quality | `promotion_rule`, `quality_gate`, `curation_rule`, `security_policy`, `license_policy` |
| Delivery & notifications | `webhook`, `signing_key`, `email_subscription` |
| Instance settings | `system_settings`, `telemetry_settings`, `format_handler` |

Data sources: `artifactkeeper_repository` (by key), `artifactkeeper_user` and
`artifactkeeper_group` (by UUID).

Not modelled yet: a few repository sub-configs (cache-TTL, routing-rules, npm-scope-policy,
pypi-tracks, upstream-auth) and some format-specific fields. MAINTAINING.md has the exact
boundary.

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

The provider version tracks the Artifact Keeper version it's validated against: `v1.6.3`
targets Artifact Keeper 1.6.3. Pin with `~> 1.6.3`.

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
