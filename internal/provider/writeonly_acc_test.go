package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// TestAccWriteOnlySecrets applies the in-place and the replacing `_wo` paths, then bumps every `_wo_version`. The empty plans after each step are what prove no secret or redacted read-back lands in state
func TestAccWriteOnlySecrets(t *testing.T) {
	config := func(version int) string {
		return fmt.Sprintf(`
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "proxied" {
  key          = "tf-acc-wo-egress"
  name         = "TF Acc WO Egress"
  format       = "npm"
  repo_type    = "remote"
  upstream_url = "https://registry.npmjs.org"
}

resource "artifactkeeper_repository_egress_proxy" "wo" {
  repository_key       = artifactkeeper_repository.proxied.key
  mode                 = "explicit"
  proxy_url_wo         = "http://svc:secret-%[1]d@proxy.internal:3128"
  proxy_url_wo_version = %[1]d
}

resource "artifactkeeper_user" "wo" {
  username            = "tf-acc-wo-user"
  email               = "tf-acc-wo-user@example.com"
  password_wo         = "Tf-acc-wo-pass-%[1]d!"
  password_wo_version = %[1]d
}

resource "artifactkeeper_remote_instance" "wo" {
  name               = "tf-acc-wo-remote"
  url                = "https://ak-wo.example.com"
  api_key_wo         = "dummy-remote-key-%[1]d"
  api_key_wo_version = %[1]d
}

resource "artifactkeeper_peer" "wo" {
  name               = "tf-acc-wo-peer"
  endpoint_url       = "https://peer-wo.example.com"
  api_key_wo         = "dummy-peer-key-%[1]d"
  api_key_wo_version = %[1]d
}
`, version)
	}
	check := func(version string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckNoResourceAttr("artifactkeeper_repository_egress_proxy.wo", "proxy_url"),
			resource.TestCheckNoResourceAttr("artifactkeeper_repository_egress_proxy.wo", "proxy_url_wo"),
			resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.wo", "proxy_credentials_configured", "true"),
			resource.TestCheckResourceAttr("artifactkeeper_repository_egress_proxy.wo", "proxy_url_wo_version", version),
			resource.TestCheckNoResourceAttr("artifactkeeper_user.wo", "password"),
			resource.TestCheckNoResourceAttr("artifactkeeper_user.wo", "password_wo"),
			resource.TestCheckNoResourceAttr("artifactkeeper_user.wo", "generated_password"),
			resource.TestCheckResourceAttr("artifactkeeper_user.wo", "password_wo_version", version),
			resource.TestCheckNoResourceAttr("artifactkeeper_remote_instance.wo", "api_key"),
			resource.TestCheckResourceAttr("artifactkeeper_remote_instance.wo", "api_key_wo_version", version),
			resource.TestCheckNoResourceAttr("artifactkeeper_peer.wo", "api_key"),
			resource.TestCheckResourceAttr("artifactkeeper_peer.wo", "api_key_wo_version", version),
		)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{
			{Config: config(1), Check: check("1")},
			{Config: config(2), Check: check("2")},
		},
	})
}
