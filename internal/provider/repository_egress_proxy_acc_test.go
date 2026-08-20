package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRepositoryEgressProxyResource walks the three modes on a remote
// repository, then imports. The interesting step is the credentialed one: the
// API redacts userinfo out of the read-back, so an empty plan afterwards is what
// proves the provider keeps the configured URL instead of writing "***" to state.
func TestAccRepositoryEgressProxyResource(t *testing.T) {
	const repo = `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "proxied" {
  key          = "tf-acc-egress"
  name         = "TF Acc Egress"
  format       = "npm"
  repo_type    = "remote"
  upstream_url = "https://registry.npmjs.org"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: repo + `
resource "artifactkeeper_repository_egress_proxy" "this" {
  repository_key = artifactkeeper_repository.proxied.key
  mode           = "explicit"
  proxy_url      = "http://proxy.internal:3128"
  no_proxy       = "localhost,.internal"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.this", "id", "tf-acc-egress"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.this", "mode", "explicit"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.this", "proxy_url", "http://proxy.internal:3128"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.this", "no_proxy", "localhost,.internal"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.this", "proxy_credentials_configured", "false"),
				),
			},
			{
				// Credentials in the URL: the server stores them, returns
				// http://***@proxy.internal:3128, and the plan must still be empty.
				Config: repo + `
resource "artifactkeeper_repository_egress_proxy" "this" {
  repository_key = artifactkeeper_repository.proxied.key
  mode           = "explicit"
  proxy_url      = "http://svc:secret@proxy.internal:3128"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.this", "proxy_url", "http://svc:secret@proxy.internal:3128"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.this", "proxy_credentials_configured", "true"),
				),
			},
			{
				Config: repo + `
resource "artifactkeeper_repository_egress_proxy" "this" {
  repository_key = artifactkeeper_repository.proxied.key
  mode           = "direct"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.this", "mode", "direct"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.this", "proxy_credentials_configured", "false"),
				),
			},
			{
				ResourceName:      "artifactkeeper_repository_egress_proxy.this",
				ImportState:       true,
				ImportStateId:     "tf-acc-egress",
				ImportStateVerify: true,
			},
		},
	})
}
