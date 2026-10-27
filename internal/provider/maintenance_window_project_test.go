package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMaintenanceWindowProjectFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/maintenance-windows" || r.Header.Get("Authorization") != "Bearer loopback-only" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		selector := r.URL.Query().Get("project_id")
		if selector != "" && selector != "unassigned" && selector != projectSourceID {
			t.Errorf("unexpected project selector %q", selector)
		}
		rows := []any{}
		if selector != "unassigned" {
			rows = append(rows, map[string]any{"id": "assigned123", "project_id": projectSourceID, "name": "Assigned", "start_date": "2026-10-28T00:00:00Z", "duration_minutes": 30, "timezone": "UTC"})
		}
		if selector != projectSourceID {
			rows = append(rows, map[string]any{"id": "unassigned123", "project_id": nil, "name": "Unassigned", "start_date": "2026-10-28T00:00:00Z", "duration_minutes": 30, "timezone": "UTC"})
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": rows, "result_info": map[string]any{"page": 1, "per_page": 100, "count": len(rows), "total_count": len(rows)}})
	}))
	defer server.Close()
	config := fmt.Sprintf(`provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
data "onlineornot_maintenance_windows" "all" {}
data "onlineornot_maintenance_windows" "assigned" { project_id = %q }
data "onlineornot_maintenance_windows" "unassigned" { project_id = "unassigned" }
`, server.URL, projectSourceID)
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config, Check: resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("data.onlineornot_maintenance_windows.all", "maintenance_windows.#", "2"),
		resource.TestCheckResourceAttr("data.onlineornot_maintenance_windows.assigned", "maintenance_windows.#", "1"),
		resource.TestCheckResourceAttr("data.onlineornot_maintenance_windows.assigned", "maintenance_windows.0.project_id", projectSourceID),
		resource.TestCheckResourceAttr("data.onlineornot_maintenance_windows.unassigned", "maintenance_windows.#", "1"),
		resource.TestCheckResourceAttr("data.onlineornot_maintenance_windows.unassigned", "maintenance_windows.0.id", "unassigned123"),
		resource.TestCheckNoResourceAttr("data.onlineornot_maintenance_windows.unassigned", "maintenance_windows.0.project_id"),
	)}}})
}

func TestMaintenanceWindowProjectConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/maintenance-windows" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]any{"code": 10000, "message": "maintenance_window_project_mismatch"}}})
	}))
	defer server.Close()
	config := fmt.Sprintf(`provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_maintenance_window" "test" {
 name = "Maintenance"
 project_id = %q
 checks = ["check1234"]
 start_date = "2026-10-28T00:00:00Z"
 duration_minutes = 30
 days_of_week = ["MONDAY"]
 timezone = "UTC"
}
`, server.URL, projectSourceID)
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile("maintenance_window_project_mismatch")}}})
}
