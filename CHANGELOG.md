# Changelog

All notable changes to this provider are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The provider version tracks
the Artifact Keeper release it is validated against (`v1.7.1` = Artifact Keeper 1.7.1);
see [MAINTAINING.md](MAINTAINING.md#versioning--releasing).

## [Unreleased]

### Added

- `artifactkeeper_migration_job`: `repo_mappings` attribute (source repo key -> target key) to
  rename repositories during migration. **Proton branch only, not released**: the backend
  capability is in neither Artifact Keeper 1.7.0 nor 1.7.1, and is pending upstream as
  [artifact-keeper#3038](https://github.com/artifact-keeper/artifact-keeper/pull/3038). It ships
  once that lands in a released backend.

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
