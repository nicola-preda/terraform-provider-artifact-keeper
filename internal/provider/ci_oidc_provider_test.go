package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCiOidcProviderTokenTTL: token_ttl_secs set at create, changed in place,
// imported, and kept when dropped from config. Requires TF_ACC and a live
// instance on 1.12.0 or later.
func TestAccCiOidcProviderTokenTTL(t *testing.T) {
	config := func(ttl string) string {
		return `
provider "artifactkeeper" {}

resource "artifactkeeper_ci_oidc_provider" "ttl" {
  name       = "tf-acc-oidc-ttl"
  issuer_url = "https://gitlab.example.com"
  ` + ttl + `
}
`
	}
	const name = "artifactkeeper_ci_oidc_provider.ttl"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("token_ttl_secs = 600"), Check: resource.TestCheckResourceAttr(name, "token_ttl_secs", "600")},
			{Config: config("token_ttl_secs = 3600"), Check: resource.TestCheckResourceAttr(name, "token_ttl_secs", "3600")},
			{ResourceName: name, ImportState: true, ImportStateVerify: true},
			{Config: config(""), PlanOnly: true},
		},
	})
}
