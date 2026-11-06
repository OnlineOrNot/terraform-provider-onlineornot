package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMultiProjectCollectionFilters(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		query := r.URL.Query()
		if !reflect.DeepEqual(query["project_id"], []string{projectSourceID, "unassigned"}) {
			t.Errorf("expected repeated project IDs, got %s", r.URL)
		}
		page := query.Get("page")
		seen[r.URL.Path+":"+page] = true
		row := map[string]any{"id": "assigned123", "project_id": projectSourceID, "name": "Assigned", "url": "https://example.com", "check_type": "UPTIME", "status": "PENDING", "method": "GET", "description": "hook", "grace_period": 1, "start_date": "2026-10-28T00:00:00Z", "duration_minutes": 30, "timezone": "UTC"}
		if page == "2" {
			row["id"] = "unassigned123"
			row["project_id"] = nil
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []any{row}, "result_info": map[string]any{"page": page, "per_page": 1, "count": 1, "total_count": 2}})
	}))
	defer server.Close()
	config := fmt.Sprintf("provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\n", server.URL)
	var checks []resource.TestCheckFunc
	for _, kind := range []string{"checks", "heartbeats", "webhooks", "maintenance_windows"} {
		config += fmt.Sprintf("data \"onlineornot_%s\" \"selected\" { project_ids = [%q, \"unassigned\"] }\n", kind, projectSourceID)
		address := "data.onlineornot_" + kind + ".selected"
		checks = append(checks, resource.TestCheckResourceAttr(address, kind+".#", "2"), resource.TestCheckResourceAttr(address, kind+".0.project_id", projectSourceID), resource.TestCheckNoResourceAttr(address, kind+".1.project_id"))
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config, Check: resource.ComposeTestCheckFunc(checks...)}}})
	mu.Lock()
	defer mu.Unlock()
	for _, path := range []string{"checks", "heartbeats", "webhooks", "maintenance-windows"} {
		for _, page := range []string{"1", "2"} {
			if !seen["/v1/"+path+":"+page] {
				t.Errorf("missing %s page %s", path, page)
			}
		}
	}
}

func TestProjectFilterValidation(t *testing.T) {
	for _, kind := range []string{"checks", "heartbeats", "webhooks", "maintenance_windows"} {
		for _, tc := range []struct{ name, filter, error string }{
			{"conflicting", fmt.Sprintf("project_id = %q\nproject_ids = [%q]", projectSourceID, projectSourceID), "Invalid Attribute Combination"},
			{"too_many", "project_ids = [" + strings.Repeat(fmt.Sprintf("%q,", projectSourceID), 101) + "]", "Invalid Attribute Value"},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				config := fmt.Sprintf("provider \"onlineornot\" {api_key=\"loopback-only\"}\ndata \"onlineornot_%s\" \"invalid\" {\n%s\n}", kind, tc.filter)
				resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config, PlanOnly: true, ExpectError: regexp.MustCompile(tc.error)}}})
			})
		}
	}
}
