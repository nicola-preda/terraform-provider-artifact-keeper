# Ordered path-rewrite rules for a remote repository's proxy requests. Rules are
# evaluated top to bottom, first match wins, so list order matters.
resource "artifactkeeper_repository_routing_rules" "npm_proxy" {
  repository_key = artifactkeeper_repository.npm_proxy.key
  rules = [
    {
      path_pattern = "^/v1/(.*)$"
      rewrite_to   = "/$1"
    },
    {
      path_pattern = "^/legacy/(.*)$"
      rewrite_to   = "/new/$1"
    },
  ]
}
