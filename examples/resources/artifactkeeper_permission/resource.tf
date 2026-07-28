# Grant a group read/write on a repository. principal_id / target_id reference
# the managed resources so Terraform creates them before the grant.
resource "artifactkeeper_permission" "platform_docker" {
  principal_type = "group"
  principal_id   = artifactkeeper_group.platform.id
  target_type    = "repository"
  target_id      = artifactkeeper_repository.docker_local.id
  actions        = ["read", "write"]
}
