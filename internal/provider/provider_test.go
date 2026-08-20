package provider

import (
	"context"
	"net/http"
	"os"
	"strings"
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
	if got := len(resp.ResourceSchemas); got != 51 {
		t.Errorf("resource schemas: want 51, got %d", got)
	}
	if got := len(resp.DataSourceSchemas); got != 4 {
		t.Errorf("data source schemas: want 4, got %d", got)
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

// testAccSkipIfEndpointMissing skips the test when the backend under test doesn't
// serve `method path`, so a test for a resource whose endpoint arrived in a later
// release reports as skipped rather than failing against an older instance.
//
// 404/405 means the router has no such route; anything else, including 401 and
// the 4xx from an empty probe body, means it does and the test should run and
// say what really broke.
//
// Pick `method` so a *sibling* route can't answer the probe: a collection path
// can be swallowed by an adjacent `/{id}` route and come back 400 rather than
// 404, in which case probe with a method the sibling doesn't have.
func testAccSkipIfEndpointMissing(t *testing.T, method, path string) {
	t.Helper()

	endpoint := strings.TrimSuffix(os.Getenv("ARTIFACT_KEEPER_ENDPOINT"), "/")
	req, err := http.NewRequest(method, endpoint+"/api/v1"+path, nil)
	if err != nil {
		t.Fatalf("building probe request: %v", err)
	}
	if token := os.Getenv("ARTIFACT_KEEPER_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("probing %s: %v", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		t.Skipf("backend does not serve %s %s (status %d); skipping", method, path, resp.StatusCode)
	}
}
