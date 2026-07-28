//go:build tools

// Package tools pins the codegen tools used by `go generate` (docs generation)
// so they're tracked in go.mod. It isn't part of the built provider.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
