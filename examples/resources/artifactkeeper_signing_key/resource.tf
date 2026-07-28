# A repository-scoped RSA signing key (defaults: key_type = "rsa", algorithm = "rsa4096").
resource "artifactkeeper_signing_key" "docker_local" {
  repository_id = artifactkeeper_repository.docker_local.id
  name          = "docker-local signing key"
}

# A GPG key for signing Debian/RPM repository metadata. OpenPGP-signed formats
# (Debian InRelease/Release.gpg, RPM repomd.xml.asc) require key_type = "gpg".
resource "artifactkeeper_signing_key" "apt" {
  repository_id = artifactkeeper_repository.apt_local.id
  name          = "apt metadata key"
  key_type      = "gpg"
  uid_name      = "Artifact Keeper"
  uid_email     = "artifacts@example.com"
}

# A global (instance-wide) RSA-2048 key, not tied to a repository.
resource "artifactkeeper_signing_key" "global" {
  name      = "global signing key"
  algorithm = "rsa2048"
}

# Signing keys are immutable; any change forces a new key. Import by key ID:
#   terraform import artifactkeeper_signing_key.docker_local <key-uuid>
