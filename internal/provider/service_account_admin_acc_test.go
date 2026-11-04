package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServiceAccountIsAdmin covers is_admin on artifactkeeper_service_account,
// which goes through GET/PATCH /users/{id} because the service-account endpoints
// carry no such flag: promote at create, demote in place, and read it back on
// import. A service account created without is_admin reads back false.
func TestAccServiceAccountIsAdmin(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test; set TF_ACC to run")
	}
	testAccPreCheck(t)

	config := func(isAdmin string) string {
		adminLine := ""
		if isAdmin != "" {
			adminLine = "is_admin = " + isAdmin
		}
		return `
provider "artifactkeeper" {}

resource "artifactkeeper_service_account" "admin" {
  name         = "tf-acc-sa-admin"
  display_name = "TF Acc admin service account"
  ` + adminLine + `
}

resource "artifactkeeper_service_account" "plain" {
  name = "tf-acc-sa-plain"
}
`
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_service_account.admin", "is_admin", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_service_account.admin", "is_active", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_service_account.plain", "is_admin", "false"),
				),
			},
			{
				ResourceName:      "artifactkeeper_service_account.admin",
				ImportState:       true,
				ImportStateVerify: true,
				// name is not returned by the API and is preserved from configuration.
				ImportStateVerifyIgnore: []string{"name"},
			},
			{
				Config: config("false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_service_account.admin", "is_admin", "false"),
				),
			},
			{
				// Dropping the attribute keeps the server-side value (Computed).
				Config:   config(""),
				PlanOnly: true,
			},
		},
	})
}
