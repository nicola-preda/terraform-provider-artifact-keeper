package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/provider"
)

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name artifactkeeper

// version is set by the release process via -ldflags; "dev" for local builds.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// Registry address. The local name used in `required_providers` is
		// "artifactkeeper" (it must match the resource type prefix); the source
		// type here is "artifact-keeper" to match the repository/binary name.
		Address: "registry.terraform.io/nicola-preda/artifact-keeper",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
