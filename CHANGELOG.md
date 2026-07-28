# Changelog

All notable changes to this provider are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The provider version tracks
the Artifact Keeper release it is validated against (`v1.6.3` = Artifact Keeper 1.6.3);
see [MAINTAINING.md](MAINTAINING.md#versioning--releasing).

## [Unreleased]

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

[Unreleased]: https://github.com/nicola-preda/terraform-provider-artifact-keeper/compare/v1.6.3...HEAD
[1.6.3]: https://github.com/nicola-preda/terraform-provider-artifact-keeper/releases/tag/v1.6.3
