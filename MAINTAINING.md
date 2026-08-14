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
| Validated against | **Artifact Keeper 1.7.4** (2026-08-14) |
| Provider changes needed | additive only (`migration_job.repo_mappings`) |
| Acceptance suite | run live against the 1.7.4 backend image (2026-08-14); all 12 `TestAcc` functions pass |

Drop-in from 1.7.1, and the cleanest bump so far. `routes.rs` is **byte-identical** between
the two tags, so no consumed endpoint moved, was added, or was removed. A mechanical diff of
every `Serialize`/`Deserialize` struct under `api/handlers/`, `services/` and `models/`
turns up exactly **two added fields across the whole API surface** and no removal, rename or
retype: `MigrationConfig.repo_mappings` (now modelled) and
`BackupManifest.artifacts_unreadable` (backup runs aren't modelled). No enum variant was
dropped either. 1.7.2 was never released; 1.7.3 carries the bulk of the work and 1.7.4 is
security-only. The churn in the diff is almost entirely package wire protocols, proxy-cache
correctness, and tests, none of which the provider touches.

1.7.3 does change **authz and behavior** on surface the provider uses, without changing any
schema. The load-bearing ones are in Caveats below: the fine-grained permissions API is now
admin-only, and the npm scope-policy payload rules relaxed. Re-check each backend bump with
the procedure below.

## Versioning & releasing

The provider version matches the Artifact Keeper version it's validated against: tag
`vX.Y.Z` means validated against Artifact Keeper X.Y.Z and equals `ValidatedUpstreamVersion`
(`internal/provider/provider.go`). This release is `v1.7.4` (AK 1.7.4); consumers pin
`~> 1.7.4`. A provider-only fix that keeps the same validated AK version is rare; if one
is needed, bump the patch ahead of AK and note it in the changelog.

Cutting a release:

1. Run the drift check below. Set `ValidatedUpstreamVersion` and the compatibility table
   to the AK version you validated against, in the same commit.
2. Tag `v<that version>` (e.g. `v1.7.0`) and push it. The `release` workflow runs
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
| `peer_instance_label` | `GET /peers/{id}/labels`, `POST`\|`DELETE …/{label_key}` (one resource per label; the whole-set `PUT` is not used) | `handlers/peer_instance_labels.rs` | `PeerLabelResponse`, `AddPeerLabelRequest` |
| `user_api_token` | `POST/GET /users/{id}/tokens`, `DELETE …/{token_id}` (admin can mint for another user) | `handlers/users.rs` | `CreateUserApiTokenRequest`, `ApiTokenResponse`, `ApiTokenCreatedResponse` |
| `age_gate` | `GET/PUT /repositories/{key}/age-gate` | `handlers/age_gate.rs`, `services/age_gate_service.rs` | `AgeGateConfigResponse`, `UpdateAgeGateConfigRequest`, `AgeGateMode` |
| (auth) | `POST /auth/login` | `handlers/auth.rs` | `LoginRequest`, `LoginResponse` |

The 29 resources added after the baseline aren't all listed here. Most map 1:1 to a
same-named backend handler (`handlers/<name>.rs`); a few share one (e.g.
`group_membership`→`groups`, `service_account_token`→`service_accounts`,
`system_settings`/`telemetry_settings`→admin settings). To find a resource's endpoints
and structs, open its `internal/client/<name>.go` (it names the exact paths), then diff
that handler on a bump.

## How to re-check drift on a version bump

When the backend moves to a new tag (say `v1.8.0`), verify the provider before
declaring compatibility. `PREV` = the tag in "Validated against" above (`v1.7.4`).

Fastest first pass, and the one that actually caught both the 1.7.1 and 1.7.4 deltas: diff every
serializable struct between the two tags, rather than reading handlers one by one. Extract
`pub struct X { … }` field lists from `api/handlers/`, `services/` and `models/` at both
tags, sort, and diff. Additions are safe; a removal, rename or retype of a field the
provider reads or sends is the breaking set.

```sh
BK=~/git/github.com/artifact-keeper/artifact-keeper
git -C "$BK" fetch --tags

# 1. Did any consumed route move? (path + HTTP method)
git -C "$BK" diff v1.7.4 v1.8.0 -- backend/src/api/routes.rs

# 2. Diff each consumed struct. Repeat per row in the map above.
git -C "$BK" diff v1.7.4 v1.8.0 -- backend/src/api/handlers/repositories.rs
git -C "$BK" diff v1.7.4 v1.8.0 -- backend/src/api/handlers/peers.rs
# … etc. Inspect a struct at the new tag with:
git -C "$BK" grep -n 'struct RepositoryResponse' v1.8.0
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
- **Migration is admin-only (1.7.0, #2603).** The `/migrations` nest moved from
  `auth_middleware` to `admin_middleware`, so `migration_source` and
  `migration_job` now need a global-admin token. Schemas are unchanged; a
  non-admin token that worked in 1.6.x now 403s on these two resources.
- **Per-repo config writes need repo-admin (1.7.0).** `update_repo_security`,
  `set_routing_rules`/`delete_routing_rules`, `put_pypi_track`/`delete_pypi_track`
  and `set_upstream_auth` now call `require_repo_admin` in addition to
  `require_repo_write_access`; repo `write` no longer suffices. No effect for a
  global-admin or repo-admin token (the usual IaC posture).
- **Token scope vocabulary enforced at mint (1.7.0, #2996).** `generate_api_token`
  now runs `validate_scopes_pure`, so `api_token`/`service_account_token` reject
  non-canonical scopes (bare `read`/`write`/…) that 1.6.x accepted. The provider
  validates the same list client-side (`tokenScopes` in `api_token_resource.go`),
  so a bad scope fails at plan time. The omitted-scopes default is now
  `read:artifacts`.
- **Age gate: `mode` and the (format, mode) enforcement check (1.7.1, #2264).** `mode` picks
  the timestamp the age is measured from: `upstream_publish_time` (server default, trusts the
  registry) or `first_seen` (local first sighting; unbackdatable, but treats any newly-cached
  version as new). Omitting it keeps the repository's current mode, so pre-mode clients that
  PUT `{enabled, min_age_days}` stay valid. **Enabling** the gate now 400s when the format
  can't enforce that mode: npm and pypi support both modes, go supports `first_seen` only;
  disabling is always allowed. 1.7.1 is also where the gate started actually being enforced
  on npm/PyPI/Go proxy downloads (a `451` below `min_age_days`), so turning it on is now
  load-bearing rather than declarative-only.
- **group_membership can't manage SSO-owned groups (1.7.0, #2874).** Add/remove
  members 409s when the group has an `external_source` (`oidc`/`saml`/`ldap`). The
  new computed `group.external_source` attribute surfaces the owner; only local
  groups are membership-manageable from Terraform.
- **The permissions API is admin-only (1.7.3, #3229).** `GET /permissions`,
  `GET /permissions/{id}` and `DELETE /permissions/{id}` gained `require_admin()`, joining the
  writes that already had it, so `artifactkeeper_permission` now needs a global-admin token for
  Read and Delete as well as Create/Update. A non-admin token that worked on 1.7.1 gets `403`.
  No schema change; the reads previously extracted no auth at all, which is what the fix closes.
- **Docker Hub pull-through ignores upstream credentials (1.7.4, GHSA-78h6-3wp8-2542).**
  Credentialed Docker Hub pull-through now authenticates anonymously. Setting
  `repository_upstream_auth` on a Docker Hub proxy no longer lifts its rate limits, and the
  advisory asks for those credentials to be rotated. The resource still writes them; the backend
  just doesn't use them on that path.
- **npm scope policy accepts an inactive payload on non-remote repos (1.7.3, #3304).** A
  relaxation: `repository_npm_scope_policy` create/update on a hosted or virtual repository no
  longer 400s on an inactive payload.
- **Storage totals change basis (1.7.3, #3134, #3249).** `total_storage_bytes` now sums the usage
  ledger, folding in OCI blob and proxy-cached bytes. Display-only, and the provider doesn't read
  it, but quota-adjacent dashboards step up on upgrade day. Quota admission is unchanged, so no
  repository managed by `artifactkeeper_repository` becomes over-quota as a result.

## Capability gaps (backend offers, provider doesn't model)

Not bugs; scope decisions. Current as of v1.7.4 (**49 resources + 4 data sources**).
The backend has ~90 handler modules; most are package wire protocols or imperative
actions that aren't IaC. Every whole-object endpoint is modelled, the per-repository
sub-config endpoints have `repository_*` resources (`repository_security`,
`repository_cache_ttl`, `repository_npm_scope_policy`, `repository_routing_rules`,
`repository_pypi_track`, `repository_upstream_auth`, `repository_release_target`), and the
v1.7.1 pass closed the last two API-only gaps.

**Closed in v1.7.4:** `migration_job.repo_mappings`, the only settable field 1.7.3 added.

**Closed in v1.7.1:** `age_gate.mode`; `peer_instance_label` (peer key/value labels, which
`sync_policy` match rules consume); `user_api_token` (admin minting a token for another
user via `POST /users/{id}/tokens`); and the `user`/`group` data sources now look up by
`username`/`name` instead of requiring a UUID you'd have to find first, joined by a new
`project` data source keyed on `key`.

**Closed in v1.7.0:** repository `quarantine_enabled`/`quarantine_duration_minutes`,
`curation_enabled`/`curation_default_action`/`curation_allow_unverified`, and the nested
`debian` proxy filter; the SSO `allow_legacy_rsa_keys` (OIDC),
`insecure_skip_verify`/`ca_certificate` (LDAP), `use_absolute_acs_url`/`map_groups_to_groups`
(SAML) fields; `group_membership` now reads all members (paged); typed `curation_rule`
(`rule_type`/`config`/`scope`).

**Intentional single-owner duplicates.** A few settings exist on both the repository object
and a dedicated resource; the sub-resource owns them, so they are deliberately not on
`artifactkeeper_repository`: create-time `upstream_username`/`upstream_password` (owned by
`repository_upstream_auth`), `npm_allowed_scopes`/`npm_allow_unscoped` (owned by
`repository_npm_scope_policy`), `release_repository_key` (owned by
`repository_release_target`), and `age_gate_mode` (owned by `age_gate`).
`npm_allowed_name_patterns`, which no sub-resource covers, *is* on the repository object.

**Correctly excluded (not IaC):** imperative actions (approval, quarantine, plugin
install, promotion/migration runs, cache invalidation, per-repo and global storage GC,
backup runs, CI token exchange), read-only/monitoring endpoints (`/admin/stats`,
`/analytics/*`, `/system/config`, `/admin/storage-backends`, `/admin/audit`), artifact- and
build-scoped records that CI produces rather than Terraform (`/builds`, `/sbom`,
`/artifacts/{id}/labels`, Dependency-Track finding analysis), package wire protocols, and
SMTP (env-configured; the UI's SMTP tab is display-and-test only, there is no save
endpoint, as its own source comments note).

**Not a provider gap, worth knowing:** the web UI ships an admin rate-limit page
(`src/lib/api/rate-limits.ts`: `GET /admin/rate-limits`, `GET/POST/DELETE
/admin/rate-limits/exemptions`) whose backend endpoints **do not exist** in 1.7.4 —
rate-limit exemptions are still env-only (`RATE_LIMIT_EXEMPT_*`). Nothing to model until
the backend side lands (web #270 / backend #680).

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

`docker-compose.test.yml` boots a minimal 1.7.4 backend (Postgres + OpenSearch + the
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

The acceptance suite is pinned to the 1.7.4 backend image and was last run live against 1.7.4
(2026-08-14): all 12 `TestAcc` functions pass. It exercises the great majority
of resources. The smoke/prereq tests create the full post-baseline set alongside
`repository`/`user`/`group`/`permission` and the data sources, including the v1.7.1
additions: `age_gate` sets `mode = "first_seen"` on an npm remote (exercising the
(format, mode) enforcement path), and `user_api_token` mints a token for the test user.
`migration_job` (and, via it, `migration_source`) are covered too:
`TestAccMigrationJobResource` creates a connection and a pending job, since neither needs a
reachable source, and sets `repo_mappings` so the v1.7.4 addition is exercised on the wire
(verified separately by hand: the job response echoes the stored map back). The client
deliberately doesn't model the response's `config` object, so no configured input is
recoverable on import; `repo_mappings` therefore sits in that test's
`ImportStateVerifyIgnore` alongside the rest of them.
Only `peer` (and therefore `peer_instance_label`), `sso_*`, `plugin`, and
actually *running* a migration (start / test-connection) sit outside it; they need external
systems (a reachable peer, an IdP, an installable WASM plugin git repo, a live source
registry) to exercise. `plugin` is source-verified against 1.7.4 (every endpoint, field, and
the `active` status string checked against `handlers/plugins.rs`) plus unit-tested, but has
not had a live end-to-end apply; `peer_instance_label` is source-verified the same way
against `handlers/peer_instance_labels.rs`.

## Client behavior

The client retries `429`/`503` honoring `Retry-After` (the backend sheds load under
concurrency), and sends `User-Agent: terraform-provider-artifact-keeper/<version>`.
