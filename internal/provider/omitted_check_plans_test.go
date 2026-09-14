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
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// Run real Terraform refresh/plan/apply against loopback, never a production API.
func TestOmittedCheckFieldsTerraformLifecycle(t *testing.T) {
	var mu sync.Mutex
	checks := map[string]map[string]any{}
	component := map[string]any{}
	drifted := false
	componentUpdates := 0
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
		var stored map[string]any
		if strings.HasPrefix(r.URL.Path, "/v1/checks/uptime") {
			id := strings.TrimPrefix(r.URL.Path, "/v1/checks/uptime/")
			if r.Method == "POST" {
				id = input["name"].(string) + "12345678"
				checks[id] = map[string]any{"id": id, "status": "ACTIVE", "check_type": "UPTIME"}
			}
			stored = checks[id]
			if stored == nil {
				t.Errorf("bad check route %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			if drifted && r.Method == "PATCH" {
				for _, key := range []string{"alert_priority", "follow_redirects", "verify_ssl", "test_interval", "timeout", "test_regions", "user_alerts", "reminder_alert_interval_minutes", "paused", "muted", "type", "version", "auth_username", "auth_password"} {
					if _, ok := input[key]; ok {
						t.Errorf("PATCH sends omitted %s: %v", key, input)
					}
				}
			}
		} else if strings.HasPrefix(r.URL.Path, "/v1/status_pages/page1234/components") {
			if r.Method != "POST" && r.URL.Path != "/v1/status_pages/page1234/components/comp1234" {
				t.Errorf("bad component route %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			stored = component
			if r.Method == "POST" {
				stored["id"] = "comp1234"
				stored["status_page_id"] = "page1234"
			}
			if drifted && r.Method == "PATCH" {
				componentUpdates++
				if _, ok := input["status"]; ok {
					t.Error("omitted live component status sent in PATCH")
				}
			}
		} else {
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		for k, v := range input {
			stored[k] = v
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
	}))
	defer server.Close()
	config := func(expected, timing, componentName, relationships string) string {
		return fmt.Sprintf(`
provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_uptime_check" "api" {
 name = "API"
 url = "https://echo.onlineornot.com/"
 method = "GET"
 headers = {Content-Type = "application/json"}
 assertions = [{type="TEXT_BODY", property="", comparison="CONTAINS", expected=%q}]
 %s
}
resource "onlineornot_uptime_check" "web" {
 name = "WEB"
 url = "https://example.com/"
}
resource "onlineornot_status_page_component" "api" {
 name = %q
 status_page_id = "page1234"
 check_ids = [%s]
}
`, server.URL, expected, timing, componentName, relationships)
	}
	api := "onlineornot_uptime_check.api"
	web := "onlineornot_uptime_check.web"
	comp := "onlineornot_status_page_component.api"
	relation := "onlineornot_uptime_check.api.id"
	unchanged := []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(web, plancheck.ResourceActionNoop),
		plancheck.ExpectResourceAction(comp, plancheck.ResourceActionNoop),
		plancheck.ExpectKnownValue(api, tfjsonpath.New("id"), knownvalue.StringExact("API12345678")),
		plancheck.ExpectKnownValue(api, tfjsonpath.New("alert_priority"), knownvalue.StringExact("LOW")),
		plancheck.ExpectKnownValue(api, tfjsonpath.New("timeout"), knownvalue.Int64Exact(45000)),
		plancheck.ExpectKnownValue(api, tfjsonpath.New("follow_redirects"), knownvalue.Bool(false)),
		plancheck.ExpectKnownValue(api, tfjsonpath.New("verify_ssl"), knownvalue.Bool(true)),
		plancheck.ExpectKnownValue(api, tfjsonpath.New("test_regions"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("US_EAST_1")})),
		plancheck.ExpectKnownValue(api, tfjsonpath.New("auth_password"), knownvalue.Null()),
		plancheck.ExpectKnownValue(api, tfjsonpath.New("version"), knownvalue.Null()),
		plancheck.ExpectKnownValue(comp, tfjsonpath.New("status"), knownvalue.StringExact("MAJOR_OUTAGE")),
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config("asdf", "", "API", relation)},
		{PreConfig: func() {
			mu.Lock()
			defer mu.Unlock()
			drifted = true
			for _, c := range checks {
				c["alert_priority"] = "LOW"
				c["timeout"] = 45000
				c["follow_redirects"] = false
				c["verify_ssl"] = true
				c["test_regions"] = []string{"US_EAST_1"}
			}
			component["status"] = "MAJOR_OUTAGE"
		}, Config: config("asdf", "", "API", relation), PlanOnly: true},
		{Config: config("changed", "", "API", relation), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: unchanged}},
		{Config: config("changed", "confirmation_period_seconds = 0\nrecovery_period_seconds = 0", "API", relation), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: unchanged}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(api, "confirmation_period_seconds", "0"), resource.TestCheckResourceAttr(api, "recovery_period_seconds", "0"))},
		{PreConfig: func() {
			mu.Lock()
			defer mu.Unlock()
			if componentUpdates != 0 {
				t.Errorf("unrelated component updated %d times", componentUpdates)
			}
		}, Config: config("changed", "confirmation_period_seconds = 0\nrecovery_period_seconds = 0", "Renamed", relation), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectKnownValue(comp, tfjsonpath.New("id"), knownvalue.StringExact("comp1234")),
			plancheck.ExpectUnknownValue(comp, tfjsonpath.New("status")),
		}}},
		{Config: config("changed", "confirmation_period_seconds = 0\nrecovery_period_seconds = 0", "Renamed", "onlineornot_uptime_check.web.id"), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectUnknownValue(comp, tfjsonpath.New("status"))}}},
	}})
	mu.Lock()
	defer mu.Unlock()
	if componentUpdates != 2 {
		t.Errorf("expected two intentional component updates, got %d", componentUpdates)
	}
}

func TestTypedMonitorOmittedDefaultsTerraformLifecycle(t *testing.T) {
	for _, kind := range []string{"dns", "tcp"} {
		t.Run(kind, func(t *testing.T) {
			var mu sync.Mutex
			stored := map[string]any{"id": "monitor1234", "status": "ACTIVE"}
			drifted := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.URL.Path != "/v1/checks/"+kind && r.URL.Path != "/v1/checks/"+kind+"/monitor1234" {
					t.Errorf("unexpected route %s", r.URL.Path)
				}
				if r.Method == "POST" || r.Method == "PATCH" {
					var input map[string]any
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						t.Error(err)
					}
					if drifted && r.Method == "PATCH" {
						for _, key := range []string{"alert_priority", "timeout", "dns_protocol", "tcp_ip_family", "tcp_should_fail", "reminder_alert_interval_minutes"} {
							if _, ok := input[key]; ok {
								t.Errorf("omitted field %s sent: %v", key, input)
							}
						}
						for _, key := range []string{"confirmation_period_seconds", "recovery_period_seconds"} {
							if input[key] != float64(0) {
								t.Errorf("explicit zero lost for %s: %v", key, input)
							}
						}
					}
					for k, v := range input {
						stored[k] = v
					}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
			}))
			defer server.Close()
			fields := "dns_domain = \"example.com\"\ndns_record_type = \"A\""
			if kind == "tcp" {
				fields = "tcp_hostname = \"example.com\"\ntcp_port = 443"
			}
			config := func(extra string) string {
				return fmt.Sprintf("provider \"onlineornot\" {\napi_key=\"loopback-only\"\nbase_url=%q\n}\nresource \"onlineornot_%s_check\" \"test\" {\nname=\"typed\"\n%s\n%s\n}", server.URL, kind, fields, extra)
			}
			address := "onlineornot_" + kind + "_check.test"
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config("")},
				{PreConfig: func() {
					mu.Lock()
					defer mu.Unlock()
					drifted = true
					stored["alert_priority"] = "LOW"
					stored["timeout"] = 45000
					if kind == "dns" {
						stored["dns_protocol"] = "TCP"
					} else {
						stored["tcp_ip_family"] = "IPv6"
						stored["tcp_should_fail"] = true
					}
				}, Config: config(""), PlanOnly: true},
				{Config: config("confirmation_period_seconds = 0\nrecovery_period_seconds = 0"), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectKnownValue(address, tfjsonpath.New("id"), knownvalue.StringExact("monitor1234")),
					plancheck.ExpectKnownValue(address, tfjsonpath.New("alert_priority"), knownvalue.StringExact("LOW")),
					plancheck.ExpectKnownValue(address, tfjsonpath.New("timeout"), knownvalue.Int64Exact(45000)),
				}}},
			}})
		})
	}
}

func TestCheckExplicitClearsTerraformLifecycle(t *testing.T) {
	server, _ := mockCheckAPI(t, "uptime")
	defer server.Close()
	config := func(fields string) string {
		return fmt.Sprintf(`provider "onlineornot" {
 api_key="loopback-only"
 base_url=%q
}
resource "onlineornot_uptime_check" "test" {
 name="clears"
 url="https://example.com"
 %s
}`, server.URL, fields)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config(`body="payload"
headers={Content-Type="application/json"}
user_alerts=["user1234"]
assertions=[{type="TEXT_BODY",property="",comparison="CONTAINS",expected="asdf"}]
follow_redirects=true
verify_ssl=true
alert_priority="LOW"`)},
		{Config: config(`body=""
headers={}
user_alerts=[]
assertions=[]
follow_redirects=false
verify_ssl=false
alert_priority="HIGH"`), Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("onlineornot_uptime_check.test", "body", ""),
			resource.TestCheckResourceAttr("onlineornot_uptime_check.test", "headers.%", "0"),
			resource.TestCheckResourceAttr("onlineornot_uptime_check.test", "user_alerts.#", "0"),
			resource.TestCheckResourceAttr("onlineornot_uptime_check.test", "assertions.#", "0"),
			resource.TestCheckResourceAttr("onlineornot_uptime_check.test", "follow_redirects", "false"),
			resource.TestCheckResourceAttr("onlineornot_uptime_check.test", "alert_priority", "HIGH"),
		)},
		// Removing Optional+Computed configuration relinquishes management; it is
		// not a clear. Values remain as read from the API.
		{Config: config(""), PlanOnly: true},
	}})
}
