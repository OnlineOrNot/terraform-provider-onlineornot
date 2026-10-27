package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const projectDefaultID = "defaultProject123456"
const projectSourceID = "sourceProject1234567"
const projectTargetID = "targetProject1234567"

func TestProjectsResourceLifecycle(t *testing.T) {
	var mu sync.Mutex
	var stored map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer loopback-only" {
			t.Error("missing bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "POST":
			var body struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			if r.URL.Path != "/v1/projects" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			stored = map[string]any{"id": projectSourceID, "name": body.Name, "is_default": false, "created_at": "2026-10-15T00:00:00Z", "updated_at": "2026-10-15T00:00:00Z"}
			w.WriteHeader(http.StatusCreated)
		case "PATCH":
			var body struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			stored["name"] = body.Name
		case "GET":
			if r.URL.Path == "/v1/projects" {
				page := 1
				row := stored
				if r.URL.Query().Get("page") == "2" {
					page = 2
					row = map[string]any{"id": projectDefaultID, "name": "Default", "is_default": true, "created_at": "2026-10-15T00:00:00Z", "updated_at": "2026-10-15T00:00:00Z"}
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []any{row}, "result_info": map[string]any{"page": page, "per_page": 1, "count": 1, "total_count": 2}})
				return
			}
		case "DELETE":
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": projectSourceID}})
			return
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
	}))
	defer server.Close()
	config := func(name string) string {
		return fmt.Sprintf(`provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_project" "test" { name = %q }
data "onlineornot_projects" "all" { depends_on = [onlineornot_project.test] }
`, server.URL, name)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config("staging"), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("onlineornot_project.test", "id", projectSourceID), resource.TestCheckNoResourceAttr("onlineornot_project.test", "is_default"), resource.TestCheckResourceAttr("data.onlineornot_projects.all", "projects.0.name", "staging"))},
		{Config: config("Default"), Check: resource.ComposeTestCheckFunc(resource.TestCheckNoResourceAttr("onlineornot_project.test", "is_default"), resource.TestCheckResourceAttr("onlineornot_project.test", "name", "Default"))},
		{ResourceName: "onlineornot_project.test", ImportState: true, ImportStateId: projectSourceID, ImportStateVerify: true},
	}})
}

func TestProjectMonitorMovesLifecycle(t *testing.T) {
	cases := []struct{ kind, endpoint, config string }{
		{"check", "/v1/checks", `url = "https://example.com"`},
		{"uptime_check", "/v1/checks/uptime", `url = "https://example.com"`},
		{"browser_check", "/v1/checks/browser", `url = "https://example.com"`},
		{"dns_check", "/v1/checks/dns", "dns_domain = \"example.com\"\ndns_record_type = \"A\""},
		{"tcp_check", "/v1/checks/tcp", "tcp_hostname = \"example.com\"\ntcp_port = 443"},
		{"heartbeat", "/v1/heartbeats", "grace_period = 1\nreport_period = 60"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			var mu sync.Mutex
			var stored map[string]any
			creates, moves, deletes := 0, 0, 0
			invalidReceipt := false
			movePath := "/v1/checks/monitor123/move"
			if tc.kind == "heartbeat" {
				movePath = "/v1/heartbeats/monitor123/move"
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
						t.Error("omitted project selector must not be sent on create")
					}
					stored = map[string]any{"id": "monitor123", "project_id": projectDefaultID, "status": "PENDING", "version": "NODE24", "dns_protocol": "UDP", "tcp_ip_family": "IPv4", "tcp_should_fail": false, "alert_priority": "HIGH", "reminder_alert_interval_minutes": 1440, "confirmation_period_seconds": 60, "recovery_period_seconds": 180, "timeout": 10000}
					for k, v := range body {
						stored[k] = v
					}
					w.WriteHeader(http.StatusCreated)
				case r.Method == "POST" && r.URL.Path == movePath:
					if len(body) != 1 {
						t.Error("move must contain only project_id")
					}
					moves++
					if invalidReceipt {
						json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": "wrong1234", "project_id": body["project_id"]}})
						return
					}
					stored["project_id"] = body["project_id"]
					json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": "monitor123", "project_id": stored["project_id"]}})
					return
				case r.Method == "PATCH" && r.URL.Path == tc.endpoint+"/monitor123":
					if body["name"] == "reject-patch" {
						w.WriteHeader(500)
						json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]any{"code": 10000, "message": "fixture patch rejected"}}})
						return
					}
					if _, ok := body["project_id"]; ok {
						t.Error("ownership must never be PATCHed")
					}
					for k, v := range body {
						stored[k] = v
					}
				case r.Method == "GET" && r.URL.Path == tc.endpoint+"/monitor123":
				case r.Method == "DELETE" && r.URL.Path == tc.endpoint+"/monitor123":
					deletes++
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
			}))
			defer server.Close()
			address := "onlineornot_" + tc.kind + ".test"
			config := func(project string) string {
				setting := ""
				if project != "" {
					setting = fmt.Sprintf("project_id = %q", project)
				}
				return fmt.Sprintf("provider \"onlineornot\" {\n api_key = \"loopback-only\"\n base_url = %q\n}\nresource \"onlineornot_%s\" \"test\" {\nname = \"monitor\"\n%s\n%s\n}", server.URL, tc.kind, tc.config, setting)
			}
			check := func(project string) resource.TestCheckFunc {
				return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(address, "id", "monitor123"), resource.TestCheckResourceAttr(address, "project_id", project))
			}
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, AdditionalCLIOptions: &resource.AdditionalCLIOptions{Plan: resource.PlanOptions{NoRefresh: true}}, Steps: []resource.TestStep{
				{Config: config(""), Check: check(projectDefaultID)},
				{Config: config(projectSourceID), Check: check(projectSourceID), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)}}},
				{Config: config(projectTargetID), Check: check(projectTargetID)},
				{Config: strings.Replace(config(projectSourceID), `name = "monitor"`, `name = "reject-patch"`, 1), ExpectError: regexp.MustCompile("fixture patch rejected")},
				{Config: config(projectSourceID), PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
				{PreConfig: func() { mu.Lock(); invalidReceipt = true; mu.Unlock() }, Config: config(projectTargetID), ExpectError: regexp.MustCompile("invalid project move acknowledgement")},
				{Config: config(projectSourceID), PlanOnly: true},
				{Config: config(""), PlanOnly: true},
				{ResourceName: address, ImportState: true, ImportStateId: "monitor123", ImportStateVerify: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			if creates != 1 || moves != 4 || deletes != 1 {
				t.Errorf("create/move/delete counts = %d/%d/%d, want 1/4/1", creates, moves, deletes)
			}
		})
	}
}

func TestProjectImmutableOwnershipPlans(t *testing.T) {
	for _, kind := range []string{"environment_variable", "webhook", "maintenance_window"} {
		t.Run(kind, func(t *testing.T) {
			var mu sync.Mutex
			var stored map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "POST" {
					json.NewDecoder(r.Body).Decode(&stored)
					stored["id"] = "owned1234"
					if _, ok := stored["enabled"]; !ok {
						stored["enabled"] = true
					}
					if stored["project_id"] != projectSourceID {
						t.Error("create lost explicit project selection")
					}
					delete(stored, "value")
				}
				if r.Method == "PATCH" {
					t.Error("nonmovable ownership must require replacement")
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
			}))
			defer server.Close()
			settings := `url = "https://example.com/hook"
events = ["uptime.down"]`
			if kind == "environment_variable" {
				settings = "name = \"API_KEY\"\ntype = \"secret\"\nvalue = \"loopback-secret\"\nvalue_version = \"1\""
			}
			if kind == "maintenance_window" {
				settings = `name = "maintenance"
start_date = "2026-10-28T00:00:00Z"
duration_minutes = 30
days_of_week = ["MONDAY"]
timezone = "UTC"`
			}
			config := func(project string) string {
				return fmt.Sprintf("provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\nresource \"onlineornot_%s\" \"test\" {\n%s\nproject_id=%q\n}", server.URL, kind, settings, project)
			}
			address := "onlineornot_" + kind + ".test"
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config(projectSourceID), Check: resource.TestCheckResourceAttr(address, "project_id", projectSourceID)},
				{Config: config(projectTargetID), PlanOnly: true, ExpectNonEmptyPlan: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate)}}},
			}})
		})
	}
}

func TestProjectCollectionFilters(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		selector := r.URL.Query().Get("project_id")
		if selector != "" && selector != projectSourceID {
			t.Errorf("unexpected selector %q", selector)
		}
		seen[r.URL.Path+":"+selector] = true
		rows := []any{map[string]any{"id": "owned1234", "project_id": projectSourceID, "name": "monitor", "url": "https://example.com", "check_type": "UPTIME_CHECK", "status": "PENDING", "method": "GET", "description": "hook", "grace_period": 1}}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": rows, "result_info": map[string]any{"page": 1, "per_page": 100, "count": 1, "total_count": 1}})
	}))
	defer server.Close()
	var config strings.Builder
	fmt.Fprintf(&config, "provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\n", server.URL)
	for _, kind := range []string{"checks", "heartbeats", "webhooks"} {
		fmt.Fprintf(&config, "data \"onlineornot_%s\" \"all\" {}\ndata \"onlineornot_%s\" \"selected\" {project_id=%q}\n", kind, kind, projectSourceID)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config.String(), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("data.onlineornot_checks.selected", "checks.0.project_id", projectSourceID), resource.TestCheckResourceAttr("data.onlineornot_heartbeats.selected", "heartbeats.0.project_id", projectSourceID), resource.TestCheckResourceAttr("data.onlineornot_webhooks.selected", "webhooks.0.project_id", projectSourceID))}}})
	mu.Lock()
	defer mu.Unlock()
	for _, p := range []string{"checks", "heartbeats", "webhooks"} {
		for _, id := range []string{"", projectSourceID} {
			if !seen["/v1/"+p+":"+id] {
				t.Errorf("missing %s filter %q", p, id)
			}
		}
	}
}

func TestProjectMoveFailurePreservesIdentity(t *testing.T) {
	var mu sync.Mutex
	moves, patches := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/move") {
			moves++
			w.WriteHeader(409)
			json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]any{"code": 10000, "message": "project_move_variable_conflict"}}})
			return
		}
		if r.Method == "PATCH" {
			patches++
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": "monitor123", "project_id": projectSourceID, "name": "monitor", "grace_period": 1, "report_period": 60, "alert_priority": "LOW", "reminder_alert_interval_minutes": 1440, "status": "PENDING"}})
	}))
	defer server.Close()
	config := func(project string) string {
		return fmt.Sprintf("provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\nresource \"onlineornot_heartbeat\" \"test\" {\nname=\"monitor\"\ngrace_period=1\nreport_period=60\nproject_id=%q\n}", server.URL, project)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config(projectSourceID)},
		{Config: config(projectTargetID), ExpectError: regexp.MustCompile("project_move_variable_conflict")},
		{Config: config(projectSourceID), PlanOnly: true},
	}})
	mu.Lock()
	defer mu.Unlock()
	if moves != 1 || patches != 0 {
		t.Errorf("move/patch counts %d/%d want 1/0", moves, patches)
	}
}
