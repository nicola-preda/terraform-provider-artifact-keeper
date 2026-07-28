resource "artifactkeeper_user" "alice" {
  username     = "alice"
  email        = "alice@example.com"
  display_name = "Alice Example"
  is_admin     = false
}

# When password is omitted, the API generates one, exposed as
# generated_password (sensitive) on the created resource.
