# Look up an existing group by name to resolve its id for memberships.
data "artifactkeeper_group" "platform" {
  name = "platform"
}
