# Look up an existing group by its UUID (from the admin UI, the groups API, or a
# managed artifactkeeper_group resource's id).
data "artifactkeeper_group" "platform" {
  id = "6a1f0b93-2e74-4c18-9d5a-7b2c4e60af31"
}
