package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestProviderSchema builds the full provider schema through the plugin protocol,
// which runs the framework's schema validation (Required/Computed conflicts, bad
// plan modifiers, …). It needs no backend, so it runs on every `go test`.
func TestProviderSchema(t *testing.T) {
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatalf("provider server: %v", err)
	}
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("schema diagnostic: %s, %s", d.Summary, d.Detail)
		}
	}
	if got := len(resp.ResourceSchemas); got != 37 {
		t.Errorf("resource schemas: want 37, got %d", got)
	}
	if got := len(resp.DataSourceSchemas); got != 3 {
		t.Errorf("data source schemas: want 3, got %d", got)
	}
}

// testAccProtoV6ProviderFactories wires the in-process provider for acceptance
// tests. The provider reads its endpoint/credentials from the environment, so the
// test configs use an empty provider block.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"artifactkeeper": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck fails fast when the live-instance env isn't configured. Acceptance
// tests only run when TF_ACC is set.
func testAccPreCheck(t *testing.T) {
	if os.Getenv("ARTIFACT_KEEPER_ENDPOINT") == "" {
		t.Fatal("ARTIFACT_KEEPER_ENDPOINT must be set for acceptance tests")
	}
	if os.Getenv("ARTIFACT_KEEPER_TOKEN") == "" &&
		(os.Getenv("ARTIFACT_KEEPER_USERNAME") == "" || os.Getenv("ARTIFACT_KEEPER_PASSWORD") == "") {
		t.Fatal("set ARTIFACT_KEEPER_TOKEN, or ARTIFACT_KEEPER_USERNAME and ARTIFACT_KEEPER_PASSWORD")
	}
}
