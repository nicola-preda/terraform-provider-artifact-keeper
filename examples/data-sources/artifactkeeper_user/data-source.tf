# Look up an existing user by username to resolve its id for memberships and
# role assignments.
data "artifactkeeper_user" "alice" {
  username = "alice"
}
