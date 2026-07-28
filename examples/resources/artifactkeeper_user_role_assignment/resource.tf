# Grant a role to a user. user_id references the managed user so Terraform
# creates it before the assignment. The assignment is not repository-scoped and
# has no mutable attributes, changing either id replaces the edge.
resource "artifactkeeper_user_role_assignment" "alice_release_manager" {
  user_id = artifactkeeper_user.alice.id
  role_id = "550e8400-e29b-41d4-a716-446655440000" # UUID of an existing role
}
