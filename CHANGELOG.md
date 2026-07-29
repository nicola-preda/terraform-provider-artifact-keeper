# Changelog

All notable changes to this provider are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The provider version tracks
the Artifact Keeper release it is validated against (`v1.6.4` = Artifact Keeper 1.6.4);
see [MAINTAINING.md](MAINTAINING.md#versioning--releasing).

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

[1.6.4]: https://github.com/nicola-preda/terraform-provider-artifact-keeper/compare/v1.6.3...v1.6.4
[1.6.3]: https://github.com/nicola-preda/terraform-provider-artifact-keeper/releases/tag/v1.6.3
