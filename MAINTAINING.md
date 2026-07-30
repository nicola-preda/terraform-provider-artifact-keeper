# Maintaining terraform-provider-artifact-keeper

How the provider maps onto the Artifact Keeper backend, and what to check when the
backend releases a new version.

## What this is

A Terraform provider (HashiCorp **terraform-plugin-framework**) that manages an
Artifact Keeper instance via its REST API at
`https://<host>/api/v1`. Two layers:

- `internal/client/`: a thin, framework-agnostic HTTP client. One file per API
  area, plus `client.go` (transport, `do()`, `APIError`, `IsNotFound`) and
  `auth.go` (`POST /auth/login`). Uses stock `encoding/json`, so **unknown
  response fields are ignored**; additive backend changes don't break it.
- `internal/provider/`: one `*_resource.go` per Terraform resource, mapping the
  client types onto framework schemas. `helpers.go` holds shared converters.

**Design principle:** Terraform attribute names mirror the API JSON field names
one-to-one (`repo_type`, `is_public`, `allow_anonymous_access`, …). Keep it that
way, no renaming, no camelCase.

## Backend = source of truth

The API is defined by the Rust backend at
`~/git/github.com/artifact-keeper/artifact-keeper` (tags `vX.Y.Z`). The provider
mirrors a **subset** of it. When the backend bumps versions, the provider does
not need changes unless a **consumed** endpoint/field changed.

- Routes: `backend/src/api/routes.rs` (all under `.nest("/api/v1", …)`).
- Request/response structs: in the per-area handler under
  `backend/src/api/handlers/`, or a service under `backend/src/services/`.
- Rust ⇒ required-ness: a bare field is **required** (client must send it or the
  request 400s); `Option<T>` or `#[serde(default)]` is **optional**.

## Compatibility status

| | |
|---|---|
| Validated against | **Artifact Keeper 1.6.4** (2026-07-29) |
| Provider changes needed | none |
| Acceptance suite | green against the live 1.6.3 image; 1.6.4 is API-identical (only the openapi version string changed), so it carries over |

Drop-in across the 1.6.x line: no consumed route, field, or type has changed. Re-check
each backend bump with the procedure below. Two runtime tightenings apply (see Caveats).

## Versioning & releasing

The provider version matches the Artifact Keeper version it's validated against: tag
`vX.Y.Z` means validated against Artifact Keeper X.Y.Z and equals `ValidatedUpstreamVersion`
(`internal/provider/provider.go`). This release is `v1.6.4` (AK 1.6.4); consumers pin
`~> 1.6.4`. A provider-only fix that keeps the same validated AK version is rare; if one
is needed, bump the patch ahead of AK and note it in the changelog.

Cutting a release:

1. Run the drift check below. Set `ValidatedUpstreamVersion` and the compatibility table
   to the AK version you validated against, in the same commit.
2. Tag `v<that version>` (e.g. `v1.6.4`) and push it. The `release` workflow runs
   GoReleaser, which builds the per-platform archives, `SHA256SUMS`, the GPG signature,
   and the registry manifest, and attaches them to a GitHub release.
3. First release only: upload the GPG public key to the Terraform Registry and add the
   provider there.

## Resource → backend map

Use this to locate the structs to diff on a version bump. Struct names are the
Rust names (provider Go names differ where noted).

| Resource | Endpoints | Backend file(s) | Structs |
|---|---|---|---|
| `repository` | `POST /repositories`, `GET/PATCH/DELETE /repositories/{key}` | `handlers/repositories.rs` | `RepositoryResponse`, `CreateRepositoryRequest`, `UpdateRepositoryRequest` |
| `api_token` | `POST/GET /profile/access-tokens`, `DELETE …/{id}` | `handlers/profile.rs`, `handlers/users.rs`, `services/auth_service.rs` | `CreateAccessTokenRequest`, `ApiTokenResponse`, `ApiTokenCreatedResponse`, `ApiTokenListResponse` (`{items:[…]}`) |
| `peer` | `POST /peers`, `GET/DELETE /peers/{id}` (no update) | `handlers/peers.rs` | `PeerInstanceResponse`, `RegisterPeerRequest` |
| `migration_source` | `POST /migrations/connections`, `GET/DELETE …/{id}` | `handlers/migration.rs` | `ConnectionResponse`, `CreateConnectionRequest`, `ConnectionCredentials` |
| `migration_job` | `POST /migrations`, `GET/DELETE …/{id}` (creates a pending job only; start/pause/cancel are imperative, not modeled) | `handlers/migration.rs` | `MigrationJobResponse`, `CreateMigrationRequest`, `MigrationConfig` |
| `plugin` | `POST /plugins/install/git`, `GET/DELETE …/{id}`, `POST …/{id}/enable`\|`disable`\|`config` (git installs only; zip/local/reload not modelled) | `handlers/plugins.rs` | `PluginResponse`, `InstallFromGitRequest`, `PluginInstallResponse`, `UpdatePluginConfigRequest` |
| `repository_security` | `GET/PUT /repositories/{key}/security` (per-repo scan config; upsert, no delete) | `handlers/security.rs`, `services/scan_config_service.rs` | `RepoSecurityResponse`/`ScanConfigResponse`, `UpsertScanConfigRequest` |
| `repository_cache_ttl` | `GET/PUT /repositories/{key}/cache-ttl` | `handlers/repositories.rs` | `CacheTtlResponse`, `SetCacheTtlRequest` |
| `repository_npm_scope_policy` | `GET/PUT /repositories/{key}/npm-scope-policy` | `handlers/repositories.rs` | `NpmScopePolicyResponse`, `SetNpmScopePolicyRequest` |
| `repository_routing_rules` | `GET/POST/DELETE /repositories/{key}/routing-rules` (POST replaces the ordered list) | `handlers/repositories.rs`, `services/routing_rules.rs` | `RoutingRulesResponse`, `SetRoutingRulesRequest`, `RoutingRule` |
| `repository_pypi_track` | `GET /…/pypi-tracks`, `PUT/DELETE /…/pypi-tracks/{project}` (per project) | `handlers/repositories.rs` | `PypiTrackResponse`, `PypiTracksListResponse`, `PypiTrackRequest` |
| `repository_upstream_auth` | `PUT /repositories/{key}/upstream-auth` (write-only, no GET) | `handlers/repositories.rs` | `UpstreamAuthRequest` |
| `repository_release_target` | `GET/PUT /promotion/repositories/{key}/release-target` | `handlers/promotion.rs` | `ReleaseTargetResponse`, `SetReleaseTargetRequest` |
| `sso_oidc` | `POST /admin/sso/oidc`, `GET/PUT/DELETE …/{id}` | `handlers/sso_admin.rs` (routes), `services/auth_config_service.rs` (structs) | `OidcConfigResponse`, `Create/UpdateOidcConfigRequest` |
| `sso_ldap` | `POST /admin/sso/ldap`, `GET/PUT/DELETE …/{id}` | same as OIDC | `LdapConfigResponse`, `Create/UpdateLdapConfigRequest` |
| `sso_saml` | `POST /admin/sso/saml`, `GET/PUT/DELETE …/{id}` | same as OIDC | `SamlConfigResponse`, `Create/UpdateSamlConfigRequest` |
| `user` | `POST /users`, `GET/PATCH/DELETE /users/{id}` | `handlers/users.rs` | `AdminUserResponse`, `CreateUserRequest`, `UpdateUserRequest`, `CreateUserResponse` |
| `group` | `POST /groups`, `GET/PUT/DELETE /groups/{id}` | `handlers/groups.rs` | `GroupResponse`, `CreateGroupRequest` (used for POST **and** PUT) |
| `permission` | `POST /permissions`, `GET/PUT/DELETE /permissions/{id}` | `handlers/permissions.rs`, `services/permission_service.rs` | `PermissionResponse`, `CreatePermissionRequest` (POST **and** PUT) |
| (auth) | `POST /auth/login` | `handlers/auth.rs` | `LoginRequest`, `LoginResponse` |

The 27 resources added after the baseline aren't all listed here. Most map 1:1 to a
same-named backend handler (`handlers/<name>.rs`); a few share one (e.g.
`group_membership`→`groups`, `service_account_token`→`service_accounts`,
`system_settings`/`telemetry_settings`→admin settings). To find a resource's endpoints
and structs, open its `internal/client/<name>.go` (it names the exact paths), then diff
that handler on a bump.

## How to re-check drift on a version bump

When the backend moves to a new tag (say `v1.7.0`), verify the provider before
declaring compatibility. `PREV` = the tag in "Validated against" above.

```sh
BK=~/git/github.com/artifact-keeper/artifact-keeper
git -C "$BK" fetch --tags

# 1. Did any consumed route move? (path + HTTP method)
git -C "$BK" diff v1.6.4 v1.7.0 -- backend/src/api/routes.rs

# 2. Diff each consumed struct. Repeat per row in the map above.
git -C "$BK" diff v1.6.4 v1.7.0 -- backend/src/api/handlers/repositories.rs
git -C "$BK" diff v1.6.4 v1.7.0 -- backend/src/api/handlers/peers.rs
# … etc. Inspect a struct at the new tag with:
git -C "$BK" grep -n 'struct RepositoryResponse' v1.7.0
```

**What counts as breaking** (needs a provider change), for a field the provider
sends or reads:

- a response field the provider reads is **removed or renamed** → Read/state breaks;
- a response field's **type changed** (e.g. `String`→`i64`, `Option<T>`→`T`);
- a **new required** request field (bare, no `#[serde(default)]`) → Create/Update 400s;
- a route path/method changed.

**Non-breaking** (safe to ignore, optionally worth exposing later): new response
fields, new `Option` request fields, widened enums.

For a big jump, fan the per-row diffs out across parallel workers.

## Caveats / behavioral notes (not schema breaks)

- **Permissions validate the principal at write time (added after v1.2.0).**
  The server `400`s if `principal_type` isn't `user`/`group`/`service_account`,
  or if `principal_id` doesn't exist as that type. The type values are enforced
  client-side too (a `OneOf` validator on `principal_type`), so a bad type fails
  at plan time. The existence check can't happen client-side: referencing a
  managed `artifactkeeper_user`/`_group` gets the apply order right, but a
  hardcoded or stale `principal_id` will 400 at apply.
- **Admin-gated token scopes.** Non-admin callers can't mint `promote:artifacts`
  or `trigger:sync`. The profile create path does not otherwise validate scope
  names (only count ≤50, length ≤256).
- **SSO vs local login.** When any SSO provider is enabled, non-admin local
  username/password login is refused. Use a token or an admin account for the
  provider credentials. (Pre-existing since v1.2.0.)
- **Peers report `offline`** until the backend receives heartbeats; there is no
  heartbeat emitter upstream yet. `status` drift is expected.
- **api_token Read treats expired-but-listed tokens as gone** (the list endpoint
  filters only `revoked_at IS NULL`). See the doc comment in
  `internal/client/api_token.go` and `api_token_test.go`.

## Capability gaps (backend offers, provider doesn't model)

Not bugs; scope decisions. Current as of v1.6.x (**46 resources + 3 data sources**).
The backend has ~90 handler modules; most are package wire protocols or imperative
actions that aren't IaC. All whole-object manageable resources are modelled, and the
per-repository sub-config endpoints now have `repository_*` resources too
(`repository_security`, `repository_cache_ttl`, `repository_npm_scope_policy`,
`repository_routing_rules`, `repository_pypi_track`, `repository_upstream_auth`,
`repository_release_target`). What's left is a few repository main-object fields:

**Repository main-object fields not yet exposed:** `quarantine_enabled` /
`quarantine_duration_minutes` (update-only), `npm_allowed_scopes` /
`npm_allow_unscoped` / `npm_allowed_name_patterns`, `debian` (proxy filter),
`release_repository_key`, upstream basic/bearer auth (username/password on create).
(`promotion_only`, `versioning_enabled`, `project_id`, `trusted_gpg_key`,
`custom_user_agent`, `apt_*`, `storage_backend`, `format_key`, `index_upstream_url`,
`pypi_upstream_index_path` and virtual-repo `members` are now modelled.)

**Other partials:**

- `api_token`: self tokens only (`/profile/access-tokens`); admin minting for
  another user (`/users/:id/tokens`) isn't covered.
- `group_membership` Read pages only the first 50 members (the shared HTTP client
  can't pass query params yet; extend `client.do` for full pagination).
- SSO: `allow_legacy_rsa_keys` (OIDC); `insecure_skip_verify`/`ca_certificate`
  (LDAP); `use_absolute_acs_url`/`map_groups_to_groups` (SAML; inconsistent, since
  OIDC exposes `map_groups_to_groups`).

**Correctly excluded (not IaC):** imperative actions (approval, quarantine, plugin
install, promotion/migration runs, CI token exchange), read-only/monitoring
endpoints, package wire protocols, and SMTP (env-configured; only a test endpoint).

Don't advertise these as "coming soon" in the README. Document what's implemented,
and add the resource in the same change that lists it.

## Build / test / local run

```sh
go build ./...
go vet ./...
go test ./...        # unit tests only; acceptance tests gate on a live instance + env var
go generate ./...    # regenerate docs/ from schema + examples (tfplugindocs)

# Run against a locally built binary via dev_overrides in ~/.terraformrc:
#   provider_installation { dev_overrides { "nicola-preda/artifact-keeper" = "<GOBIN>" } direct {} }
```

`docs/` is generated from the schema descriptions and the files under `examples/`:
edit those and run `go generate ./...`; don't hand-edit `docs/`.

## Acceptance tests

`docker-compose.test.yml` boots a minimal 1.6.3 backend (Postgres + OpenSearch + the
pinned backend image; `ADMIN_PASSWORD=admin`, `JWT_SECRET` must be ≥32 chars). Run:

```sh
docker compose -f docker-compose.test.yml up -d
# First boot: admin is admin/admin and must rotate the password before the API
# unlocks. Rotate, then mint an API token and use it; token auth avoids the login
# rate limiter (10 / 15 min per user+IP), which the many per-step logins otherwise trip.

TF_ACC=1 \
  ARTIFACT_KEEPER_ENDPOINT=http://localhost:8080 \
  ARTIFACT_KEEPER_TOKEN=<admin token> \
  TF_ACC_TERRAFORM_PATH="$(command -v tofu)" \
  TF_ACC_PROVIDER_HOST=registry.opentofu.org \
  TF_ACC_PROVIDER_NAMESPACE=hashicorp \
  go test ./internal/provider -run TestAcc -v

docker compose -f docker-compose.test.yml down -v
```

`TF_ACC_PROVIDER_HOST`/`TF_ACC_PROVIDER_NAMESPACE` are **required under OpenTofu**:
without them terraform-plugin-testing registers the provider under the legacy `-`
namespace on `registry.terraform.io`, which `tofu` rejects.

Coverage validated against live 1.6.3: the acceptance suite exercises the great majority
of resources. The smoke/prereq tests create the full post-baseline set alongside
`repository`/`user`/`group`/`permission` and the data sources. `migration_job` (and, via
it, `migration_source`) are covered too: `TestAccMigrationJobResource` creates a connection
and a pending job, since neither needs a reachable source. Only `peer`, `sso_*`, `plugin`, and
actually *running* a migration (start / test-connection) sit outside it; they need external
systems (a reachable peer, an IdP, an installable WASM plugin git repo, a live source
registry) to exercise. `plugin` is source-verified against 1.6.4 (every endpoint, field, and
the `active` status string checked against `handlers/plugins.rs`) plus unit-tested, but has
not had a live end-to-end apply.

## Client behavior

The client retries `429`/`503` honoring `Retry-After` (the backend sheds load under
concurrency), and sends `User-Agent: terraform-provider-artifact-keeper/<version>`.
