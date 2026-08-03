terraform {
  required_providers {
    # Local name "artifactkeeper" must match the resource type prefix
    # (artifactkeeper_*). The source type is "artifact-keeper".
    artifactkeeper = {
      source  = "nicola-preda/artifact-keeper"
      version = "~> 1.7.0"
    }
  }
}

provider "artifactkeeper" {
  endpoint = "https://artifact-keeper.example.com" # or ARTIFACT_KEEPER_ENDPOINT
  token    = var.artifact_keeper_token             # or ARTIFACT_KEEPER_TOKEN

  # Alternatively, log in with credentials instead of a token:
  # username = var.artifact_keeper_username
  # password = var.artifact_keeper_password
}
