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
)

const browserScript = "import { test, expect } from '@playwright/test';\n\ntest('example', async ({ page }) => {\n  await page.goto('https://example.com');\n  await expect(page).toHaveTitle(/Example/);\n});\n"

// Model the real contract: mutation responses have no R2 script bytes, GET
// hydrates them, and scripted checks store a NULL timeout and may have NULL URL.
func mockCheckAPI(t *testing.T, kind string) (*httptest.Server, func(string)) {
	t.Helper()
	var mu sync.Mutex
	checks := make(map[string]map[string]any)
	nextID := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		base := "/v1/checks"
		if kind != "" {
			base += "/" + kind
		}
		if r.URL.Path != base && !strings.HasPrefix(r.URL.Path, base+"/fixture") {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, base+"/")
		if r.Method == "POST" {
			nextID++
			id = "fixture"
			if nextID > 1 {
				id = fmt.Sprintf("fixture-%d", nextID)
			}
		}
		stored := checks[id]
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "POST", "PATCH":
			var input map[string]any
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if r.Method == "PATCH" && kind != "" {
				if _, ok := input["type"]; ok {
					t.Error("typed PATCH must omit type")
					http.Error(w, "type rejected", 400)
					return
				}
			}
			script, _ := input["script"].(string)
			if script != "" {
				if _, ok := input["timeout"]; ok {
					t.Error("script timeout must be omitted")
					http.Error(w, "timeout rejected", 400)
					return
				}
				if input["url"] == "" {
					t.Error("empty URL is invalid")
					http.Error(w, "url rejected", 400)
					return
				}
			} else if input["url"] == nil || input["url"] == "" {
				t.Error("URL required without script")
				http.Error(w, "url required", 400)
				return
			}
			if r.Method == "POST" {
				stored = make(map[string]any)
				checks[id] = stored
			}
			for k, v := range input {
				stored[k] = v
			}
			stored["id"] = id
			checkType := "UPTIME"
			if kind == "browser" || stored["type"] == "BROWSER_CHECK" {
				checkType = "BROWSER"
			}
			stored["check_type"] = checkType
			if script != "" {
				stored["timeout"] = nil
			}
			result := make(map[string]any)
			for k, v := range stored {
				if k != "script" {
					result[k] = v
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": result})
		case "GET":
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
		case "DELETE":
			delete(checks, id)
			json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	return server, func(script string) { mu.Lock(); defer mu.Unlock(); checks["fixture"]["script"] = script }
}

func TestScriptedCheckTerraformLifecycle(t *testing.T) {
	for _, resourceType := range []string{"browser_check", "check"} {
		t.Run(resourceType, func(t *testing.T) {
			kind := "browser"
			if resourceType == "check" {
				kind = ""
			}
			server, drift := mockCheckAPI(t, kind)
			defer server.Close()
			address := "onlineornot_" + resourceType + ".test"
			config := func(script string) string {
				return fmt.Sprintf(`
provider "onlineornot" {
 api_key = "fixture-key"
 base_url = %q
}
resource "onlineornot_%s" "test" {
 name = "script fixture"
 type = "BROWSER_CHECK"
 script = %q
}`, server.URL, resourceType, script)
			}
			check := func(script string) resource.TestCheckFunc {
				return resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "script", script),
					resource.TestCheckNoResourceAttr(address, "url"),
					resource.TestCheckNoResourceAttr(address, "timeout"),
				)
			}
			updated := strings.ReplaceAll(browserScript, "Example/", "Example Domain/")
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: config(browserScript), Check: check(browserScript)},
					{Config: config(browserScript), PlanOnly: true},
					{Config: config(updated), Check: check(updated)},
					{RefreshState: true, Check: check(updated)},
					{PreConfig: func() { drift(browserScript) }, Config: config(updated), PlanOnly: true, ExpectNonEmptyPlan: true},
					{Config: config(updated), Check: check(updated)},
				},
			})
		})
	}
}

func TestScriptedCheckRejectsTimeout(t *testing.T) {
	for _, resourceType := range []string{"browser_check", "check"} {
		t.Run(resourceType, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("invalid config must not call API")
				http.Error(w, "unexpected", 500)
			}))
			defer server.Close()
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
				Config: fmt.Sprintf(`provider "onlineornot" {
 api_key = "fixture-key"
base_url = %q
}
resource "onlineornot_%s" "test" {
 name = "script fixture"
type = "BROWSER_CHECK"
script = %q
timeout = 120000
}`, server.URL, resourceType, browserScript),
				PlanOnly: true, ExpectError: regexp.MustCompile("Timeout is incompatible with scripted browser checks"),
			}}})
		})
	}
}

func TestURLCheckTerraformLifecycle(t *testing.T) {
	for _, resourceType := range []string{"browser_check", "uptime_check", "check"} {
		t.Run(resourceType, func(t *testing.T) {
			kind := strings.TrimSuffix(resourceType, "_check")
			if resourceType == "check" {
				kind = ""
			}
			server, _ := mockCheckAPI(t, kind)
			defer server.Close()
			address := "onlineornot_" + resourceType + ".test"
			config := func(timeout string) string {
				return fmt.Sprintf(`provider "onlineornot" {
 api_key = "fixture-key"
base_url = %q
}
resource "onlineornot_%s" "test" {
 name = "URL fixture"
url = "https://example.com"
%s
}`, server.URL, resourceType, timeout)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config(""), Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr(address, "timeout", "10000"), resource.TestCheckResourceAttr(address, "url", "https://example.com"))},
				{Config: config("timeout = 120000"), Check: resource.TestCheckResourceAttr(address, "timeout", "120000")},
			}})
		})
	}
}

func TestBrowserCheckURLToScript(t *testing.T) {
	server, _ := mockCheckAPI(t, "browser")
	defer server.Close()
	config := func(fields string) string {
		return fmt.Sprintf(`
provider "onlineornot" {
 api_key = "fixture-key"
 base_url = %q
}
resource "onlineornot_browser_check" "test" {
 name = "conversion fixture"
 %s
}`, server.URL, fields)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config(`url = "https://example.com"`), Check: resource.TestCheckResourceAttr("onlineornot_browser_check.test", "timeout", "10000")},
		{Config: config(fmt.Sprintf("url = null\ntimeout = null\nscript = %q", browserScript)), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckNoResourceAttr("onlineornot_browser_check.test", "url"),
			resource.TestCheckNoResourceAttr("onlineornot_browser_check.test", "timeout"),
			resource.TestCheckResourceAttr("onlineornot_browser_check.test", "script", browserScript),
		)},
	}})
}

func TestScriptedCheckUnknownTimeoutResolvesToNull(t *testing.T) {
	for _, resourceType := range []string{"browser_check", "check"} {
		t.Run(resourceType, func(t *testing.T) {
			kind := "browser"
			if resourceType == "check" {
				kind = ""
			}
			server, _ := mockCheckAPI(t, kind)
			defer server.Close()
			address := "onlineornot_" + resourceType + ".test"
			config := fmt.Sprintf(`
provider "onlineornot" {
 api_key = "fixture-key"
 base_url = %q
}
resource "terraform_data" "settings" {
 input = { timeout = null }
}
resource "onlineornot_%s" "test" {
 name = "unknown timeout fixture"
 type = "BROWSER_CHECK"
 script = %q
 timeout = terraform_data.settings.output.timeout
}`, server.URL, resourceType, browserScript)
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: config, Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(address, "script", browserScript),
						resource.TestCheckNoResourceAttr(address, "timeout"),
					)},
					{Config: config, PlanOnly: true},
				},
			})
		})
	}
}
