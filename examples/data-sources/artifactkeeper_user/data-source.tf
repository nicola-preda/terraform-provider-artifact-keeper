# Look up an existing user by its UUID (from the admin UI, the users API, or a
# managed artifactkeeper_user resource's id).
data "artifactkeeper_user" "alice" {
  id = "9d4e2f7a-1c8b-4a35-b6e0-3f9a1c2d8e50"
}
