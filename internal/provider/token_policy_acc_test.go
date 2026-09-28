package provider

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTokenPolicyResource exercises the singleton through an enforcing
// policy and back out again. Safe to enforce here, unlike the TOTP policy: the
// policy is evaluated only at mint time, so the token the provider is already
// authenticating with is unaffected.
func TestAccTokenPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccSkipIfEndpointMissing(t, http.MethodGet, "/admin/settings/token-policy")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_token_policy" "this" {
  require_expiration = true
  min_days           = 7
  max_days           = 180
  default_days       = 90
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_token_policy.this", "id", "token_policy"),
					resource.TestCheckResourceAttr("artifactkeeper_token_policy.this", "require_expiration", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_token_policy.this", "min_days", "7"),
					resource.TestCheckResourceAttr("artifactkeeper_token_policy.this", "max_days", "180"),
					resource.TestCheckResourceAttr("artifactkeeper_token_policy.this", "default_days", "90"),
					// Opt-in, so CI credentials stay exempt unless asked.
					resource.TestCheckResourceAttr("artifactkeeper_token_policy.this", "apply_to_service_accounts", "false"),
					// No API_TOKEN_EXPIRATION_* in the test environment, so the
					// stored setting is in force and the resource owns it.
					resource.TestCheckResourceAttr("artifactkeeper_token_policy.this", "source", "database"),
					resource.TestCheckResourceAttr("artifactkeeper_token_policy.this", "editable", "true"),
				),
			},
			{
				ResourceName:      "artifactkeeper_token_policy.this",
				ImportState:       true,
				ImportStateId:     "token_policy",
				ImportStateVerify: true,
			},
			{
				// Dropping default_days must read back as null, not as the
				// previous value: the whole policy object is replaced on write.
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_token_policy" "this" {
  require_expiration = false
  min_days           = 1
  max_days           = 90
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_token_policy.this", "require_expiration", "false"),
					resource.TestCheckNoResourceAttr("artifactkeeper_token_policy.this", "default_days"),
				),
			},
		},
	})
}
