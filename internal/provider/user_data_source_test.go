package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource_byEmail(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_byEmail(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.onlineornot_user.test", "id"),
					resource.TestCheckResourceAttrSet("data.onlineornot_user.test", "email"),
					resource.TestCheckResourceAttrSet("data.onlineornot_user.test", "role"),
				),
			},
		},
	})
}

func TestAccUsersDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersDataSourceConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.onlineornot_users.test", "users.#"),
				),
			},
		},
	})
}

func testAccUserDataSourceConfig_byEmail() string {
	// First get all users, then look up the first one by email
	return `
data "onlineornot_users" "all" {}

data "onlineornot_user" "test" {
  email = data.onlineornot_users.all.users[0].email
}
`
}

func testAccUsersDataSourceConfig_basic() string {
	return `
data "onlineornot_users" "test" {}
`
}
