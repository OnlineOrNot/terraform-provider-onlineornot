package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCheckResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCheckResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onlineornot_check.test", "name", rName),
					resource.TestCheckResourceAttr("onlineornot_check.test", "url", "https://example.com"),
					resource.TestCheckResourceAttr("onlineornot_check.test", "method", "GET"),
					resource.TestCheckResourceAttrSet("onlineornot_check.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "onlineornot_check.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccCheckResourceConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onlineornot_check.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("onlineornot_check.test", "url", "https://example.org"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccCheckResource_withAlerts(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceConfig_withUserAlerts(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onlineornot_check.test", "name", rName),
					resource.TestCheckResourceAttrSet("onlineornot_check.test", "user_alerts.#"),
				),
			},
		},
	})
}

func testAccCheckResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "onlineornot_check" "test" {
  name = %[1]q
  url  = "https://example.com"
}
`, name)
}

func testAccCheckResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "onlineornot_check" "test" {
  name = "%[1]s-updated"
  url  = "https://example.org"
}
`, name)
}

func testAccCheckResourceConfig_withUserAlerts(name string) string {
	return fmt.Sprintf(`
data "onlineornot_users" "all" {}

resource "onlineornot_check" "test" {
  name        = %[1]q
  url         = "https://example.com"
  user_alerts = [data.onlineornot_users.all.users[0].id]
}
`, name)
}
