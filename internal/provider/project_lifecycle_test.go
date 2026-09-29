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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// These fixtures model the candidate API contract, not server isolation or
// scheduling correctness. No live credentials or remote resources are used.
func TestProjectOwnershipLifecycle(t *testing.T) {
	const source = "sourceProject12345"
	const target = "targetProject12345"
	for _, kind := range []string{"check", "uptime_check", "browser_check", "dns_check", "tcp_check", "heartbeat"} {
		for _, status := range []string{"ACTIVE", "PAUSED", "DISABLED"} {
			t.Run(kind+"/"+status, func(t *testing.T) {
				var mu sync.Mutex
				var stored map[string]any
				moves := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					var input map[string]any
					if r.Method == "POST" || r.Method == "PATCH" {
						if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
							t.Error(err)
						}
					}
					if strings.HasSuffix(r.URL.Path, "/move") {
						expected := "/v1/checks/monitor12345678/move"
						if kind == "heartbeat" {
							expected = "/v1/heartbeats/monitor12345678/move"
						}
						if r.Method != "POST" || r.URL.Path != expected || len(input) != 1 || input["project_id"] != target {
							t.Errorf("invalid move: %s %s %v", r.Method, r.URL.Path, input)
						}
						stored["project_id"] = target
						moves++
						json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": stored["id"], "project_id": target}})
						return
					}
					if r.Method == "POST" {
						if _, ok := input["project_id"]; ok {
							t.Error("omitted creation project must not be serialized")
						}
						stored = map[string]any{"id": "monitor12345678", "project_id": source, "status": status, "check_type": "UPTIME", "alert_priority": "HIGH", "reason": "fixture-reason", "ping_url": "https://example.invalid/ping/unchanged"}
					}
					if kind == "browser_check" {
						stored["check_type"] = "BROWSER"
					}
					if r.Method == "PATCH" {
						for _, key := range []string{"project_id", "paused", "muted"} {
							if _, ok := input[key]; ok {
								t.Errorf("ownership-only update sends %s", key)
							}
						}
					}
					for k, v := range input {
						stored[k] = v
					}
					json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
				}))
				defer server.Close()
				settings := `url = "https://example.com"`
				switch kind {
				case "dns_check":
					settings = "dns_domain = \"example.com\"\ndns_record_type = \"A\""
				case "tcp_check":
					settings = "tcp_hostname = \"example.com\"\ntcp_port = 443"
				case "heartbeat":
					settings = "grace_period = 60\nreport_period = 60"
				}
				config := func(project string) string {
					return fmt.Sprintf("provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\nresource \"onlineornot_%s\" \"test\" {\nname=\"fixture\"\n%s\n%s\n}", server.URL, kind, settings, project)
				}
				address := "onlineornot_" + kind + ".test"
				resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
					{Config: config(""), Check: resource.TestCheckResourceAttr(address, "project_id", source)},
					{Config: config("project_id = \"" + target + "\""), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(address, "project_id", target), resource.TestCheckResourceAttr(address, "id", "monitor12345678"), func(_ *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if moves != 1 || stored["status"] != status || stored["reason"] != "fixture-reason" || stored["ping_url"] != "https://example.invalid/ping/unchanged" {
							return fmt.Errorf("move did not preserve fixture identity/state")
						}
						return nil
					})},
					{Config: config(""), PlanOnly: true},
				}})
			})
		}
	}
}

func TestProjectCRUDLifecycle(t *testing.T) {
	const id = "projectFixture1234"
	var mu sync.Mutex
	name := "staging"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" || r.Method == "PATCH" {
			var input map[string]any
			json.NewDecoder(r.Body).Decode(&input)
			if len(input) != 1 {
				t.Error("only name may be written")
			}
			name = input["name"].(string)
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": id, "name": name, "is_default": false, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"}})
	}))
	defer server.Close()
	config := func(name string) string {
		return fmt.Sprintf("provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\nresource \"onlineornot_project\" \"test\" {name=%q}", server.URL, name)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config("staging"), Check: resource.TestCheckResourceAttr("onlineornot_project.test", "id", id)}, {Config: config("production"), Check: resource.TestCheckResourceAttr("onlineornot_project.test", "name", "production")}, {ResourceName: "onlineornot_project.test", ImportState: true, ImportStateId: id, ImportStateVerify: true}}})
}

func TestProjectDataSourcesLifecycle(t *testing.T) {
	const project = "projectFixture1234"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("data sources must only read: %s", r.Method)
		}
		var result []map[string]any
		switch r.URL.Path {
		case "/v1/projects":
			result = []map[string]any{{"id": project, "name": "staging", "is_default": false, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"}}
		case "/v1/checks", "/v1/heartbeats":
			result = []map[string]any{{"id": "monitor12345678", "name": "staging", "project_id": project}}
			if r.URL.Query().Get("project_id") == "" {
				result = append(result, map[string]any{"id": "monitor87654321", "name": "other", "project_id": "anotherProject1234"})
			} else if r.URL.Query().Get("project_id") != project {
				t.Error("unexpected filter")
			}
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": result, "result_info": map[string]int{"total_count": len(result)}})
	}))
	defer server.Close()
	config := fmt.Sprintf(`provider "onlineornot" {
 api_key="loopback-only"
 base_url=%q
}
data "onlineornot_projects" "all" {}
data "onlineornot_checks" "all" {}
data "onlineornot_checks" "filtered" {project_id=%q}
data "onlineornot_heartbeats" "all" {}
data "onlineornot_heartbeats" "filtered" {project_id=%q}
`, server.URL, project, project)
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config, Check: resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("data.onlineornot_projects.all", "projects.0.id", project),
		resource.TestCheckResourceAttr("data.onlineornot_checks.all", "checks.#", "2"),
		resource.TestCheckResourceAttr("data.onlineornot_checks.filtered", "checks.#", "1"),
		resource.TestCheckResourceAttr("data.onlineornot_checks.filtered", "checks.0.project_id", project),
		resource.TestCheckResourceAttr("data.onlineornot_heartbeats.all", "heartbeats.#", "2"),
		resource.TestCheckResourceAttr("data.onlineornot_heartbeats.filtered", "heartbeats.#", "1"),
		resource.TestCheckResourceAttr("data.onlineornot_heartbeats.filtered", "heartbeats.0.project_id", project),
	)}}})
}
