package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// Exercise the real create/update asymmetry using only a loopback API.
func TestStatusPageDefaultsLifecycle(t *testing.T) {
	var mu sync.Mutex
	var stored map[string]any
	password := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "POST":
			var input map[string]any
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if r.URL.Path == "/v1/status_pages" {
				stored = map[string]any{"id": "page1234", "name": input["name"], "subdomain": input["subdomain"], "description": nil, "allowed_ips": nil, "hide_from_search_engines": false}
			} else if r.URL.Path == "/v1/status_pages/page1234" {
				for _, key := range []string{"name", "subdomain", "description", "allowed_ips", "hide_from_search_engines"} {
					if v, ok := input[key]; ok {
						stored[key] = v
					}
				}
			} else {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			stored["custom_domain"] = nil
			if v, ok := input["custom_domain"].(string); ok {
				stored["custom_domain"] = normalizedStatusPageDomain(v)
			}
			if v, ok := input["password"].(string); ok {
				password = v
			}
		case "GET":
			if stored["description"] == "" {
				stored["description"] = nil
			}
			if ips, ok := stored["allowed_ips"].([]any); ok && len(ips) == 0 {
				stored["allowed_ips"] = nil
			}
		case "DELETE":
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
	}))
	defer server.Close()
	config := func(settings string) string {
		return fmt.Sprintf(`
provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_status_page" "test" {
 name = "defaults"
 subdomain = "defaults"
 %s
}`, server.URL, settings)
	}
	address := "onlineornot_status_page.test"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config(""), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectKnownValue(address, tfjsonpath.New("description"), knownvalue.Null()),
				plancheck.ExpectKnownValue(address, tfjsonpath.New("custom_domain"), knownvalue.Null()),
				plancheck.ExpectKnownValue(address, tfjsonpath.New("allowed_ips"), knownvalue.Null()),
			}}},
			{Config: config(`description = "hello"
allowed_ips = ["192.0.2.1"]
custom_domain = "https://status.example.com/"
password = "secret"
hide_from_search_engines = true`)},
			{Config: config(`description = ""
allowed_ips = []
hide_from_search_engines = false`), Check: func(_ *terraform.State) error {
				mu.Lock()
				defer mu.Unlock()
				if password != "" {
					return fmt.Errorf("removed password remains active")
				}
				return nil
			}},
			{Config: config(""), PlanOnly: true, ExpectNonEmptyPlan: true},
			{Config: config("")},
			{ResourceName: address, ImportState: true, ImportStateId: "page1234", ImportStateVerify: true},
		},
	})
}

// Plan-only tests never invoke an API, and prove that defaults are visible
// before apply rather than merely assigned in Create.
func TestExplicitAPIDefaultPlans(t *testing.T) {
	common := map[string]knownvalue.Check{
		"alert_priority":                  knownvalue.StringExact("LOW"),
		"confirmation_period_seconds":     knownvalue.Int64Exact(60),
		"recovery_period_seconds":         knownvalue.Int64Exact(180),
		"reminder_alert_interval_minutes": knownvalue.Int64Exact(1440),
		"timeout":                         knownvalue.Int64Exact(10000),
	}
	cases := []struct {
		kind, config string
		expected     map[string]knownvalue.Check
	}{
		{"check", "name = \"test\"\nurl = \"https://example.com\"", map[string]knownvalue.Check{"method": knownvalue.StringExact("GET"), "follow_redirects": knownvalue.Bool(true), "verify_ssl": knownvalue.Bool(false), "type": knownvalue.StringExact("UPTIME_CHECK")}},
		{"uptime_check", "name = \"test\"\nurl = \"https://example.com\"", common},
		{"browser_check", "name = \"test\"\nurl = \"https://example.com\"", map[string]knownvalue.Check{"type": knownvalue.StringExact("BROWSER_CHECK"), "timeout": knownvalue.Int64Exact(10000)}},
		{"dns_check", "name = \"test\"\ndns_domain = \"example.com\"\ndns_record_type = \"A\"", map[string]knownvalue.Check{"dns_protocol": knownvalue.StringExact("UDP")}},
		{"tcp_check", "name = \"test\"\ntcp_hostname = \"example.com\"\ntcp_port = 443", map[string]knownvalue.Check{"tcp_ip_family": knownvalue.StringExact("IPv4"), "tcp_should_fail": knownvalue.Bool(false)}},
		{"heartbeat", "name = \"test\"\ngrace_period = 1", map[string]knownvalue.Check{"alert_priority": knownvalue.StringExact("LOW"), "reminder_alert_interval_minutes": knownvalue.Int64Exact(1440), "grace_period": knownvalue.Int64Exact(1)}},
		{"maintenance_window", "name = \"test\"\ndays_of_week = [\"MONDAY\"]\nduration_minutes = 5\nstart_date = \"12:00\"\ntimezone = \"UTC\"", map[string]knownvalue.Check{"checks": knownvalue.ListExact([]knownvalue.Check{}), "heartbeats": knownvalue.ListExact([]knownvalue.Check{})}},
		{"status_page_component", "name = \"test\"\nstatus_page_id = \"page1234\"", map[string]knownvalue.Check{"status": knownvalue.StringExact("OPERATIONAL"), "display_uptime": knownvalue.Bool(true), "display_metrics": knownvalue.Bool(true)}},
		{"status_page_incident", "title = \"test\"\ndescription = \"test\"\nstatus = \"INVESTIGATING\"\nstatus_page_id = \"page1234\"", map[string]knownvalue.Check{"notify_subscribers": knownvalue.Bool(true)}},
		{"status_page_scheduled_maintenance", "title = \"test\"\ndescription = \"test\"\nstart_date = \"2030-01-01T00:00:00Z\"\nduration_minutes = 5\nstatus_page_id = \"page1234\"\nnotifications = {}", map[string]knownvalue.Check{"notifications": knownvalue.ObjectExact(map[string]knownvalue.Check{"an_hour_before": knownvalue.Bool(false), "at_start": knownvalue.Bool(true), "at_end": knownvalue.Bool(true)})}},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			var checks []plancheck.PlanCheck
			for field, expected := range tc.expected {
				checks = append(checks, plancheck.ExpectKnownValue("onlineornot_"+tc.kind+".test", tfjsonpath.New(field), expected))
			}
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
				Config:   fmt.Sprintf("provider \"onlineornot\" {\n api_key = \"loopback-only\"\n base_url = \"http://127.0.0.1:1\"\n}\nresource \"onlineornot_%s\" \"test\" {\n%s\n}", tc.kind, tc.config),
				PlanOnly: true, ExpectNonEmptyPlan: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: checks},
			}}})
		})
	}
}

func TestCheckZeroPeriodsLifecycle(t *testing.T) {
	server, _ := mockCheckAPI(t, "")
	defer server.Close()
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
		Config: fmt.Sprintf(`provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_check" "test" {
 name = "zero-periods"
 url = "https://example.com"
 confirmation_period_seconds = 0
 recovery_period_seconds = 0
 reminder_alert_interval_minutes = -1
 follow_redirects = false
 verify_ssl = false
}`, server.URL),
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("onlineornot_check.test", "confirmation_period_seconds", "0"),
			resource.TestCheckResourceAttr("onlineornot_check.test", "recovery_period_seconds", "0"),
			resource.TestCheckResourceAttr("onlineornot_check.test", "reminder_alert_interval_minutes", "-1"),
			resource.TestCheckResourceAttr("onlineornot_check.test", "follow_redirects", "false"),
		),
	}}})
}
