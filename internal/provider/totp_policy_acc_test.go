package provider

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTotpPolicyResource exercises the singleton at `disabled`, which is the
// only value CI can assert on: tightening the policy requires the calling admin
// to have TOTP enrolled, and the acceptance admin deliberately does not. So this
// covers the wiring (write, read-back, computed source/editable, empty plan,
// import) and leaves the lockout guard to the backend's own tests.
func TestAccTotpPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccSkipIfEndpointMissing(t, http.MethodGet, "/admin/settings/totp-policy")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_totp_policy" "this" {
  policy = "disabled"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_totp_policy.this", "id", "totp_policy"),
					resource.TestCheckResourceAttr("artifactkeeper_totp_policy.this", "policy", "disabled"),
					// No TOTP_POLICY in the test environment, so the stored
					// setting is what's in force and the resource owns it.
					resource.TestCheckResourceAttr("artifactkeeper_totp_policy.this", "source", "database"),
					resource.TestCheckResourceAttr("artifactkeeper_totp_policy.this", "editable", "true"),
				),
			},
			{
				ResourceName:      "artifactkeeper_totp_policy.this",
				ImportState:       true,
				ImportStateId:     "totp_policy",
				ImportStateVerify: true,
			},
		},
	})
}
