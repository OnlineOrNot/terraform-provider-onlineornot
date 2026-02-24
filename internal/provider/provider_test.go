package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestMain enables test sweepers and other test setup
func TestMain(m *testing.M) {
	resource.TestMain(m)
}

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"onlineornot": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck verifies the ONLINEORNOT_API_KEY environment variable is set
func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("ONLINEORNOT_API_KEY"); v == "" {
		t.Fatal("ONLINEORNOT_API_KEY must be set for acceptance tests")
	}
}
