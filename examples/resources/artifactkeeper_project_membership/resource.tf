# Grant a group read + write on every repository in a project.
resource "artifactkeeper_project_membership" "team_rw" {
  project_id     = artifactkeeper_project.platform.id
  principal_type = "group"
  principal_id   = artifactkeeper_group.platform_team.id
  actions        = ["read", "write"]
}

# Grant a single user read-only access to the same project.
resource "artifactkeeper_project_membership" "auditor_ro" {
  project_id     = artifactkeeper_project.platform.id
  principal_type = "user"
  principal_id   = artifactkeeper_user.auditor.id
  actions        = ["read"]
}
