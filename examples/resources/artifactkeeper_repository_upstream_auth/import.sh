# Addressed by repository key. Credentials can't be read back, so they stay
# unset after import until the next apply re-sends them.
terraform import artifactkeeper_repository_upstream_auth.npm npm-remote
