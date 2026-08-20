# Changelog

All notable changes to this provider are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The provider version tracks
the Artifact Keeper release it is validated against (`v1.8.0` = Artifact Keeper 1.8.0);
see [MAINTAINING.md](MAINTAINING.md#versioning--releasing).

## [Unreleased]

Validated against Artifact Keeper 1.8.0. A big release (40 commits, 109 backend source files)
that is nonetheless a drop-in: a mechanical diff of every serializable struct finds 227 added
fields and **no removal, rename or retype**, the enum diff finds no dropped variant, and the
only route that disappears is a package wire protocol path. Two of the endpoints 1.8.0 adds
are declarative config and are modelled here; the rest is proxy-scan visibility, the TOTP
enrollment exchange, and client-shape aliases for cargo and Helm.

### Added

- `artifactkeeper_repository_egress_proxy`: how one remote repository reaches its upstream.
  `mode` is `inherit` (follow the process-wide proxy environment), `direct` (bypass it) or
  `explicit` (this repository's own `proxy_url` and `no_proxy`). The per-repository setting
  replaces `HTTP_PROXY`/`NO_PROXY` for that repository rather than merging with them. The API
  returns `proxy_url` with credentials redacted to `***` and the URL normalised, so a
  configured value is kept verbatim and `proxy_credentials_configured` is the observable half.
  Destroy resets the repository to `inherit` rather than leaving an egress control behind.
- `artifactkeeper_totp_policy`: the system-wide 2FA enforcement policy (`disabled`,
  `required_for_admins`, `required_for_all`), plus computed `source` and `editable`. Two ways
  an apply fails with `409`, both deliberate: `TOTP_POLICY` in the environment pins the policy,
  and *tightening* it requires the admin the provider authenticates as to have TOTP enrolled
  already. Relaxing is never refused. Enrollment counts are in the response but not modelled;
  they're monitoring data that moves on its own.

### Changed

- A login that can't complete because the account owes a TOTP challenge or enrollment now fails
  with an error naming 2FA and pointing at token auth. 1.8.0 answers such a login with `200` and
  an empty `access_token`, which previously surfaced as a bare "no access_token was returned".

### Upgrade notes

- **`repository_security.block_on_policy_violation` means something new.** 1.8.0 adds an inline
  severity gate on proxy and OCI pulls (#3243/#3246) and derives it from fields this provider
  already writes, with no schema change to signal it. The flag is read as an opt-in to
  *thresholding*, so `false` now blocks on **any** finding; an absent scan-config row reads the
  same way, which reaches virtual repositories that never had one. A repository with
  `scan_on_proxy = true` and the flag off can start refusing artifacts on upgrade day off an
  unchanged Terraform config.
- **`artifactkeeper_peer_network_profile` does not work against any released backend**, and did
  not against 1.7.4 either. It sends `PUT /peers/{id}/network-profile`, which is where the
  handler's doc comment, its OpenAPI annotation and its unit test all place it, but `routes.rs`
  merges the router into the `/peers` nest without the `/:id` prefix. The only mounted path is
  `PUT /api/v1/peers/network-profile`, served by a handler that extracts a peer id from a route
  that has none. Needs an upstream fix; the resource is inert until then.

### Coverage

Every one of the 455 documented endpoints on 1.8.0 was classified, and all 143 request structs
were field-diffed against the client, with nothing left unclassified — see
[MAINTAINING.md](MAINTAINING.md#capability-gaps-backend-offers-provider-doesnt-model). The
acceptance suite was run live against the 1.8.0 image: 14 pass, 0 fail.

## [1.7.4] - 2026-08-14

Validated against Artifact Keeper 1.7.4, with the acceptance suite run live against that
backend image. The quietest bump yet on the provider's side: `routes.rs` is byte-identical
between 1.7.1 and 1.7.4, and a struct-level diff of the entire API surface finds two added
fields and nothing removed, renamed or retyped. One of the two is `repo_mappings`, which this
release ships. (There is no 1.7.2 upstream; 1.7.3 carries the work and 1.7.4 is
security-only.)

### Added

- `artifactkeeper_migration_job`: `repo_mappings` attribute, a source repo key -> target key
  map that renames repositories as they migrate (Artifact Keeper 1.7.3, #3035). Sources you
  don't list keep their name, and a renamed member of a migrated virtual repository keeps its
  membership. An existing target is reused and the source's artifacts merge into it. Pair it
  with `include_repos`, which lists the *source* keys. Changing the map replaces the job, like
  the rest of the migration config.

  Targets are validated like a hand-created repository key, and two sources may not share one
  destination. **Both checks run when the job runs, not when it is created**, so a bad map
  applies cleanly and then aborts the migration; `terraform apply` is not where you find out.
  Verified against 1.7.4: `POST /migrations` returns `201` for a duplicate target and for a
  malformed target key alike.

### Notes on the 1.7.3 backend, for anyone upgrading

No provider change was needed for these, but they change what the same config does:

- **The fine-grained permissions API is admin-only** (#3229). `GET /permissions`,
  `GET /permissions/{id}` and `DELETE /permissions/{id}` now require admin, joining the writes.
  A non-admin token that managed `artifactkeeper_permission` will start getting `403` on read
  and destroy. The usual IaC posture (a global-admin token) is unaffected.
- **`artifactkeeper_repository_npm_scope_policy` accepts an inactive payload on non-remote
  repositories** (#3304), where create/update previously rejected it.
- **Docker Hub pull-through now authenticates anonymously** even when upstream credentials are
  set, a consequence of the GHSA-78h6 fix. If you set `artifactkeeper_repository_upstream_auth`
  on a Docker Hub proxy to lift rate limits, it no longer has that effect, and the advisory asks
  you to rotate those credentials.
- **Storage totals step up on upgrade day** (#3134, #3249): OCI blob and proxy-cached bytes now
  count. Display-only, quota admission is unchanged, but alerts on storage growth will fire.

## [1.7.1] - 2026-08-07

Validated against Artifact Keeper 1.7.1, with the acceptance suite run live against that
backend image. A struct-level diff of the whole 1.7.0 → 1.7.1 API surface shows additions
only: nothing the provider reads or sends was removed, renamed, or retyped, so this is a
drop-in upgrade. Alongside the one new backend field, this release closes the last two
API-only coverage gaps and makes the identity data sources usable without knowing a UUID
up front.

### Added

- `artifactkeeper_age_gate`: `mode` attribute (Artifact Keeper 1.7.1, #2264). Chooses the
  timestamp the age is measured from, `upstream_publish_time` (server default) or
  `first_seen`. Omit it to keep the repository's current mode. Note that **enabling** the
  gate now requires a format that can enforce the chosen mode: npm and pypi support both,
  go supports `first_seen` only.
- `artifactkeeper_peer_instance_label`: key/value labels on a peer instance, which
  `artifactkeeper_sync_policy` match rules select on. One resource per label, so peers
  managed elsewhere keep their own labels; the backend re-evaluates sync policies on write.
- `artifactkeeper_user_api_token`: mints an API token for a specific user
  (`POST /users/{id}/tokens`), which an admin can do on another user's behalf. Complements
  `artifactkeeper_api_token` (the caller's own token) and
  `artifactkeeper_service_account_token` (machine identities). `scopes` is required here,
  since this endpoint has no server-side default.
- `artifactkeeper_project` data source, looking a project up by `key` to resolve the UUID
  that `repository.project_id` and project memberships need.

### Changed

- `artifactkeeper_user` and `artifactkeeper_group` data sources now look up by `username`
  and `name` respectively, rather than requiring the UUID you were probably using the data
  source to find. `id` becomes a computed attribute. **Breaking for existing configs** that
  passed `id`: swap `id = "<uuid>"` for `username = "..."` / `name = "..."`.
- The acceptance suite covers the new surface: `age_gate` exercises `mode = "first_seen"`
  on an npm remote (hitting the (format, mode) enforcement path), and `user_api_token`
  mints a token for the test user.

## [1.7.0] - 2026-07-31

Validated against Artifact Keeper 1.7.0. No consumed route, field, or type was removed or
retyped, so the provider is drop-in from 1.6.x; this release adds the new settable surface and
closes several long-standing coverage gaps.

### Added

- Per-repository sub-config resources: `artifactkeeper_repository_security` (scan config:
  `scan_enabled`, `scan_on_upload`, `scan_on_proxy`, `block_on_policy_violation`,
  `severity_threshold`), `artifactkeeper_repository_cache_ttl` (proxy cache TTL),
  `artifactkeeper_repository_npm_scope_policy` (npm scope allow-list),
  `artifactkeeper_repository_routing_rules` (ordered proxy rewrite rules),
  `artifactkeeper_repository_pypi_track` (PEP 708 tracks, per project),
  `artifactkeeper_repository_upstream_auth` (upstream credentials; write-only, no read-back), and
  `artifactkeeper_repository_release_target` (staging-to-release link).
- `artifactkeeper_repository`: `curation_enabled` and `curation_default_action` (curation-rule
  enforcement on proxy paths), `curation_allow_unverified` (keyless RPM curation-sync opt-in,
  write-only), `quarantine_enabled` / `quarantine_duration_minutes` (upload quarantine hold),
  `npm_allowed_name_patterns` (npm glob allow-list), and a nested `debian` block (remote proxy
  distribution/component/architecture filter).
- `artifactkeeper_curation_rule`: typed rules via `rule_type` (`pattern`, `publisher_trust`,
  `popularity`), a JSON `config` for the engine parameters, and `scope` (`repository` / `global`).
- `artifactkeeper_repository_security`: `proxy_scan_action` (`fail_open` / `fail_closed`);
  `severity_threshold` now also accepts `info`.
- `artifactkeeper_sso_oidc`: `allow_legacy_rsa_keys`. `artifactkeeper_sso_ldap`:
  `insecure_skip_verify`, `ca_certificate` (write-only) / `has_ca_certificate`.
  `artifactkeeper_sso_saml`: `use_absolute_acs_url`, `map_groups_to_groups`.
- `artifactkeeper_group` (resource and data source): computed `external_source` (the SSO provider
  that owns the group, or null for a local group).
- Client-side scope-vocabulary validation on `artifactkeeper_api_token`,
  `artifactkeeper_service_account_token`, and `artifactkeeper_repo_token`, so an invalid scope
  fails at plan time rather than as a 400 at apply (the backend now enforces the vocabulary).
- `artifactkeeper_project_membership` resource: a user/group access grant (with its `actions`)
  on a project, inherited by every repository assigned to the project.

### Changed

- `artifactkeeper_group_membership` now reads all members (pages past the first 50).
- `artifactkeeper_sso_oidc` sends `attribute_mapping` authoritatively on update (removed keys are
  now deleted rather than merged).
- Corrected the `artifactkeeper_api_token` default-scope docs (`read` -> `read:artifacts`).

### Fixed

- `artifactkeeper_webhook` now captures the server-generated signing `secret` into state. The
  API returns it only once, on create; it was previously discarded and unrecoverable.
- `artifactkeeper_repository_upstream_auth` now reads back the repository's `configured` and
  `configured_auth_type`, so upstream credentials cleared out of band are detected as drift.

### Upgrade notes (Artifact Keeper 1.7.0 behavior changes)

- `artifactkeeper_migration_source` and `artifactkeeper_migration_job` now require an **admin**
  token: the `/migrations` API is admin-gated (was any authenticated user).
- Several per-repository writes now require repo-**admin** (repo `write` no longer suffices):
  `artifactkeeper_repository_security`, `_repository_routing_rules`, `_repository_pypi_track`,
  `_repository_upstream_auth`.
- `artifactkeeper_group_membership` cannot manage SSO-owned groups (the API returns 409); the new
  `external_source` attribute identifies them.

## [1.6.4] - 2026-07-29

Validated against Artifact Keeper 1.6.4.

### Added

- `artifactkeeper_migration_job` resource: create a migration job (in `pending` state) against a
  `migration_source`, scoped to repositories with `include_repos`. Terraform creates the job;
  starting, pausing, and cancelling it stay manual/imperative operations. Cached remote (proxy)
  artifacts are excluded unless `include_cached_remote` is set.
- `artifactkeeper_plugin` resource: install a WASM plugin from a git source and manage its
  enabled state and JSON `config`. Only git installs are modelled; zip/local uploads and reload
  stay manual.

## [1.6.3]

First public release. Validated against Artifact Keeper 1.6.3.

### Added

- Provider configuration via `endpoint` + `token` (or `username`/`password`), each
  overridable by `ARTIFACT_KEEPER_*` environment variables.
- 37 resources:
  - Repositories: `repository` (incl. `virtual` repos with ordered `members`,
    plus promotion/versioning/project/APT/GPG/upstream config),
    `repository_label`, `repository_signing_config`, `lifecycle_policy`, `age_gate`.
  - Projects: `project`.
  - Identity & access: `user`, `group`, `group_membership`, `user_role_assignment`,
    `permission`, `service_account`, `service_account_token`, `api_token`, `repo_token`.
  - SSO / auth: `sso_oidc`, `sso_ldap`, `sso_saml`, `ci_oidc_provider`,
    `ci_oidc_identity_mapping`.
  - Replication & migration: `peer`, `peer_repository_subscription`,
    `peer_network_profile`, `remote_instance`, `sync_policy`, `migration_source`.
  - Promotion & quality: `promotion_rule`, `quality_gate`, `curation_rule`,
    `security_policy`, `license_policy`.
  - Delivery & notifications: `webhook`, `signing_key`, `email_subscription`.
  - Instance settings: `system_settings`, `telemetry_settings`, `format_handler`.
- 3 data sources: `repository`, `user`, `group`.
- Import support on every resource.

[Unreleased]: https://github.com/nicola-preda/terraform-provider-artifact-keeper/compare/v1.7.0...HEAD
[1.7.0]: https://github.com/nicola-preda/terraform-provider-artifact-keeper/compare/v1.6.4...v1.7.0
[1.6.4]: https://github.com/nicola-preda/terraform-provider-artifact-keeper/compare/v1.6.3...v1.6.4
[1.6.3]: https://github.com/nicola-preda/terraform-provider-artifact-keeper/releases/tag/v1.6.3
