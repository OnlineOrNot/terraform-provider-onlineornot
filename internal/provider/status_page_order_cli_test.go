package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// UnitTest runs Terraform against loopback only, without production credentials.
func TestOrderTerraformLifecycle(t *testing.T) {
	for _, factory := range orderFactories {
		r := factory().(*StatusPageOrderResource)
		t.Run(r.kind, func(t *testing.T) {
			var mu sync.Mutex
			m := &orderMock{kind: r.kind, ids: []string{"first", "second"}}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { mu.Lock(); defer mu.Unlock(); m.serve(t, w, req) }))
			defer server.Close()
			group := ""
			importID := "page1234"
			if r.kind == "group_components" {
				group = `group_id = "group123"`
				importID += "/group123"
			}
			config := func(ids string) string {
				return fmt.Sprintf(`
provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_%s" "test" {
 status_page_id = "page1234"
 %s
 %s = [%s]
}`, server.URL, r.name, group, r.idsKey, ids)
			}
			address := "onlineornot_" + r.name + ".test"
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: config(`"second", "first"`), Check: resource.TestCheckResourceAttr(address, r.idsKey+".0", "second")},
					{Config: config(`"first", "second"`), Check: resource.TestCheckResourceAttr(address, r.idsKey+".0", "first")},
					{ResourceName: address, ImportState: true, ImportStateId: importID, ImportStateVerify: true},
					{PreConfig: func() { mu.Lock(); defer mu.Unlock(); m.ids = []string{"second", "first"} }, Config: config(`"first", "second"`), PlanOnly: true, ExpectNonEmptyPlan: true},
					{Config: config(`"first", "second"`), Check: resource.TestCheckResourceAttr(address, r.idsKey+".0", "first")},
					{PreConfig: func() { mu.Lock(); defer mu.Unlock(); m.ids = []string{} }, Config: config(``), Check: resource.TestCheckResourceAttr(address, r.idsKey+".#", "0")},
				},
			})
			mu.Lock()
			defer mu.Unlock()
			if !slices.Equal(m.ids, []string{}) {
				t.Fatalf("destroy changed layout: %v", m.ids)
			}
		})
	}
}
