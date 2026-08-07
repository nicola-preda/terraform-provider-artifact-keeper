# Look up an existing project by key to resolve its id for repository.project_id.
data "artifactkeeper_project" "platform" {
  key = "platform"
}
