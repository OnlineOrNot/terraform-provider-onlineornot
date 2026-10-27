package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestProjectNullableOwnershipLifecycle(t *testing.T) {
	cases := []struct {
		kind, endpoint, settings, field, initial, updated string
		movable                                           bool
	}{
		{"check", "/v1/checks", `url = "https://example.com"`, "name", "monitor", "renamed", true},
		{"uptime_check", "/v1/checks/uptime", `url = "https://example.com"`, "name", "monitor", "renamed", true},
		{"browser_check", "/v1/checks/browser", `url = "https://example.com"`, "name", "monitor", "renamed", true},
		{"dns_check", "/v1/checks/dns", "dns_domain = \"example.com\"\ndns_record_type = \"A\"", "name", "monitor", "renamed", true},
		{"tcp_check", "/v1/checks/tcp", "tcp_hostname = \"example.com\"\ntcp_port = 443", "name", "monitor", "renamed", true},
		{"heartbeat", "/v1/heartbeats", "grace_period = 1\nreport_period = 60", "name", "monitor", "renamed", true},
		{"maintenance_window", "/v1/maintenance-windows", `start_date = "2026-10-28T00:00:00Z"
duration_minutes = 30
days_of_week = ["MONDAY"]
timezone = "UTC"`, "name", "maintenance", "renamed", false},
		{"webhook", "/v1/webhooks", "url = \"https://example.com/hook\"\nevents = [\"uptime.down\"]", "description", "receiver", "renamed receiver", false},
		{"environment_variable", "/v1/env", "type = \"secret\"\nvalue = \"loopback-write-only-value\"\nvalue_version = \"1\"", "name", "API_KEY", "RENAMED_KEY", false},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			var mu sync.Mutex
			var stored map[string]any
			creates, updates, moves, deletes := 0, 0, 0, 0
			movePath := "/v1/checks/owned1234/move"
			if tc.kind == "heartbeat" {
				movePath = "/v1/heartbeats/owned1234/move"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Header.Get("Authorization") != "Bearer loopback-only" {
					t.Error("missing bearer token")
				}
				var body map[string]any
				if r.Method == "POST" || r.Method == "PATCH" {
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
				}
				switch {
				case r.Method == "POST" && r.URL.Path == tc.endpoint:
					creates++
					if _, ok := body["project_id"]; ok {
						t.Error("omitted project assignment must not be sent on creation")
					}
					stored = map[string]any{"id": "owned1234", "project_id": nil, "status": "PENDING", "version": "NODE24", "dns_protocol": "UDP", "tcp_ip_family": "IPv4", "tcp_should_fail": false, "alert_priority": "HIGH", "reminder_alert_interval_minutes": 1440, "confirmation_period_seconds": 60, "recovery_period_seconds": 180, "timeout": 10000, "enabled": true}
					for key, value := range body {
						if key != "value" {
							stored[key] = value
						}
					}
					w.WriteHeader(http.StatusCreated)
				case r.Method == "POST" && r.URL.Path == movePath && tc.movable:
					moves++
					if len(body) != 1 || body["project_id"] != projectTargetID {
						t.Errorf("unexpected project move request: %v", body)
					}
					stored["project_id"] = body["project_id"]
				case r.Method == "PATCH" && r.URL.Path == tc.endpoint+"/owned1234":
					updates++
					if _, ok := body["project_id"]; ok {
						t.Error("ordinary updates must never change assignment")
					}
					if _, ok := body["value"]; ok {
						t.Error("metadata update must not resend the secret")
					}
					for key, value := range body {
						stored[key] = value
					}
				case r.Method == "GET" && r.URL.Path == tc.endpoint+"/owned1234":
				case r.Method == "DELETE" && r.URL.Path == tc.endpoint+"/owned1234":
					deletes++
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
			}))
			defer server.Close()
			address := "onlineornot_" + tc.kind + ".test"
			config := func(value, project string) string {
				return fmt.Sprintf("provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\nresource \"onlineornot_%s\" \"test\" {\n%s\n%s=%q\n%s\n}", server.URL, tc.kind, tc.settings, tc.field, value, project)
			}
			unassigned := resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(address, "id", "owned1234"), resource.TestCheckNoResourceAttr(address, "project_id"), resource.TestCheckNoResourceAttr(address, "value"))
			ignore := []string{}
			if tc.kind == "environment_variable" {
				ignore = append(ignore, "value_version")
			}
			steps := []resource.TestStep{
				{Config: config(tc.initial, ""), Check: unassigned},
				{Config: config(tc.updated, "project_id = null"), Check: unassigned, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)}}},
				{ResourceName: address, ImportState: true, ImportStateId: "owned1234", ImportStateVerify: true, ImportStateVerifyIgnore: ignore},
				{Config: config(tc.updated, ""), PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
				{PreConfig: func() { mu.Lock(); stored["project_id"] = projectSourceID; mu.Unlock() }, RefreshState: true, Check: resource.TestCheckResourceAttr(address, "project_id", projectSourceID)},
				{Config: config(tc.updated, "project_id = null"), PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
				{ResourceName: address, ImportState: true, ImportStateId: "owned1234", ImportStateVerify: true, ImportStateVerifyIgnore: ignore},
				{PreConfig: func() { mu.Lock(); delete(stored, "project_id"); mu.Unlock() }, RefreshState: true, Check: unassigned},
				{Config: config(tc.updated, ""), PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
			}
			if tc.movable {
				steps = append(steps, resource.TestStep{Config: config(tc.updated, fmt.Sprintf("project_id = %q", projectTargetID)), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(address, "id", "owned1234"), resource.TestCheckResourceAttr(address, "project_id", projectTargetID)), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)}}})
			}
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: steps})
			mu.Lock()
			defer mu.Unlock()
			wantedMoves, wantedUpdates := 0, 1
			if tc.movable {
				wantedMoves, wantedUpdates = 1, 2
			}
			if creates != 1 || deletes != 1 || moves != wantedMoves || updates != wantedUpdates {
				t.Errorf("create/update/move/delete counts = %d/%d/%d/%d, want 1/%d/%d/1", creates, updates, moves, deletes, wantedUpdates, wantedMoves)
			}
		})
	}
}

func TestProjectNullableCollectionOwnership(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "GET" || r.URL.Query().Has("project_id") || (r.URL.Path != "/v1/checks" && r.URL.Path != "/v1/heartbeats" && r.URL.Path != "/v1/webhooks") {
			t.Errorf("unexpected collection request %s %s", r.Method, r.URL)
		}
		rows := []any{
			map[string]any{"id": "omitted123", "name": "legacy", "url": "https://example.com", "check_type": "UPTIME_CHECK", "status": "PENDING", "method": "GET", "grace_period": 1},
			map[string]any{"id": "null12345", "project_id": nil, "name": "unassigned", "url": "https://example.com", "check_type": "UPTIME_CHECK", "status": "PENDING", "method": "GET", "grace_period": 1},
			map[string]any{"id": "named1234", "project_id": projectSourceID, "name": "assigned", "url": "https://example.com", "check_type": "UPTIME_CHECK", "status": "PENDING", "method": "GET", "grace_period": 1},
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": rows, "result_info": map[string]any{"page": 1, "per_page": 100, "count": 3, "total_count": 3}})
	}))
	defer server.Close()
	var config strings.Builder
	fmt.Fprintf(&config, "provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\n", server.URL)
	var checks []resource.TestCheckFunc
	for _, kind := range []string{"checks", "heartbeats", "webhooks"} {
		fmt.Fprintf(&config, "data \"onlineornot_%s\" \"all\" {project_id = null}\n", kind)
		address := "data.onlineornot_" + kind + ".all"
		checks = append(checks, resource.TestCheckResourceAttr(address, kind+".#", "3"), resource.TestCheckNoResourceAttr(address, kind+".0.project_id"), resource.TestCheckNoResourceAttr(address, kind+".1.project_id"), resource.TestCheckResourceAttr(address, kind+".2.project_id", projectSourceID))
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config.String(), Check: resource.ComposeTestCheckFunc(checks...)}}})
}

func TestProjectMetadataLifecycle(t *testing.T) {
	var mu sync.Mutex
	stored := map[string]any{"id": projectSourceID, "name": "staging", "created_at": "2026-10-16T00:00:00Z", "updated_at": "2026-10-16T00:00:00Z"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v1/projects" && r.URL.Path != "/v1/projects/"+projectSourceID {
			t.Errorf("unexpected project path %s", r.URL.Path)
		}
		if r.Method == "GET" && r.URL.Path == "/v1/projects" {
			rows := []any{stored, map[string]any{"id": projectTargetID, "name": "ordinary", "is_default": false, "created_at": "2026-10-16T00:00:00Z", "updated_at": "2026-10-16T00:00:00Z"}, map[string]any{"id": projectDefaultID, "name": "Default", "is_default": true, "created_at": "2026-10-16T00:00:00Z", "updated_at": "2026-10-16T00:00:00Z"}}
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": rows, "result_info": map[string]any{"page": 1, "per_page": 100, "count": 3, "total_count": 3}})
			return
		}
		if r.Method == "POST" || r.Method == "PATCH" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body) != 1 {
				t.Errorf("project writes must contain only name: %v", body)
			}
			stored["name"] = body["name"]
		}
		if r.Method == "DELETE" {
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": projectSourceID}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
	}))
	defer server.Close()
	config := func(name string) string {
		return fmt.Sprintf("provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\nresource \"onlineornot_project\" \"test\" {name=%q}\ndata \"onlineornot_projects\" \"all\" {depends_on=[onlineornot_project.test]}\n", server.URL, name)
	}
	check := resource.ComposeTestCheckFunc(resource.TestCheckNoResourceAttr("onlineornot_project.test", "is_default"), resource.TestCheckNoResourceAttr("data.onlineornot_projects.all", "projects.0.is_default"), resource.TestCheckNoResourceAttr("data.onlineornot_projects.all", "projects.1.is_default"), resource.TestCheckNoResourceAttr("data.onlineornot_projects.all", "projects.2.is_default"))
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config("staging"), Check: check},
		{Config: config("renamed"), Check: check},
		{ResourceName: "onlineornot_project.test", ImportState: true, ImportStateId: projectSourceID, ImportStateVerify: true},
		{Config: config("renamed"), PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
	}})
}
