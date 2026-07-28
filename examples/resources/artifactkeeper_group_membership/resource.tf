# Add a single user to a group. A membership is one (group, user) edge:
# declare one artifactkeeper_group_membership resource per member. group_id /
# user_id reference the managed resources so Terraform creates them first.
# Adding a member is idempotent server-side.
resource "artifactkeeper_group_membership" "alice_platform" {
  group_id = artifactkeeper_group.platform.id
  user_id  = artifactkeeper_user.alice.id
}
