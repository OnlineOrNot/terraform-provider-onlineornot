package provider

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

type domainExpiryAPI struct {
	server *httptest.Server
	mu     sync.Mutex
	checks map[string]map[string]any
	// Request bodies by method, in arrival order.
	bodies map[string][]map[string]any
}

var domainExpiryCheckTypes = map[string]string{"": "UPTIME", "uptime": "UPTIME", "browser": "BROWSER", "tcp": "TCP", "dns": "DNS"}

// Mirrors the API rule: enabled alerts need a target with a registered domain.
func domainExpiryTargetProblem(check map[string]any) string {
	if check["domain_expiry_alerts"] != true {
		return ""
	}
	if script, _ := check["script"].(string); script != "" {
		return "Domain expiry alerts need a check target with a registered domain; scripted browser checks have none."
	}
	target, _ := check["dns_domain"].(string)
	if hostname, _ := check["tcp_hostname"].(string); hostname != "" {
		target = hostname
	}
	if raw, _ := check["url"].(string); raw != "" {
		if parsed, err := url.Parse(raw); err == nil {
			target = parsed.Hostname()
		}
	}
	if net.ParseIP(target) != nil || target == "localhost" || strings.HasSuffix(target, ".herokuapp.com") {
		return fmt.Sprintf("Domain expiry alerts need a check target with a registered domain. %s has none.", target)
	}
	return ""
}

func newDomainExpiryAPI(t *testing.T) *domainExpiryAPI {
	t.Helper()
	api := &domainExpiryAPI{checks: map[string]map[string]any{}, bodies: map[string][]map[string]any{}}
	api.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		defer api.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		segments := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/checks"), "/")[1:]
		kind := ""
		if len(segments) > 0 {
			if _, ok := domainExpiryCheckTypes[segments[0]]; ok && segments[0] != "" {
				kind, segments = segments[0], segments[1:]
			}
		}
		var input map[string]any
		if r.Method == "POST" || r.Method == "PATCH" {
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			api.bodies[r.Method] = append(api.bodies[r.Method], input)
		}
		var id string
		switch {
		case r.Method == "POST" && len(segments) == 0:
			id = "check-fixture"
			if kind != "" {
				id = kind + "-fixture"
			}
			api.checks[id] = map[string]any{"id": id, "status": "UP", "check_type": domainExpiryCheckTypes[kind], "domain_expiry_alerts": false}
		case len(segments) == 1 && api.checks[segments[0]] != nil:
			id = segments[0]
		default:
			t.Errorf("unexpected route %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Method == "DELETE" {
			delete(api.checks, id)
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": id}})
			return
		}
		next := map[string]any{}
		for k, v := range api.checks[id] {
			next[k] = v
		}
		for k, v := range input {
			next[k] = v
		}
		if problem := domainExpiryTargetProblem(next); problem != "" {
			if r.Method == "POST" {
				delete(api.checks, id)
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []map[string]any{{"code": 1002, "message": problem}}})
			return
		}
		api.checks[id] = next
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": next})
	}))
	return api
}

func (api *domainExpiryAPI) lastBody(method string) map[string]any {
	api.mu.Lock()
	defer api.mu.Unlock()
	bodies := api.bodies[method]
	if len(bodies) == 0 {
		return nil
	}
	return bodies[len(bodies)-1]
}

func (api *domainExpiryAPI) expectLastBody(method string, expected any) resource.TestCheckFunc {
	return func(*terraform.State) error {
		body := api.lastBody(method)
		if value, ok := body["domain_expiry_alerts"]; !ok || value != expected {
			return fmt.Errorf("expected last %s to send domain_expiry_alerts=%v, got %#v", method, expected, body)
		}
		return nil
	}
}

func domainExpiryConfig(baseURL, resourceType, target, setting string) string {
	return fmt.Sprintf(`
provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_%s" "test" {
 name = "domain-expiry"
 %s
 %s
}`, baseURL, resourceType, target, setting)
}

var domainExpiryTargets = map[string]string{
	"check":         `url = "https://api.example.com/health"`,
	"uptime_check":  `url = "https://api.example.com/health"`,
	"browser_check": `url = "https://www.example.com/"`,
	"tcp_check":     "tcp_hostname = \"mail.example.com\"\n tcp_port = 443",
	"dns_check":     "dns_domain = \"example.com\"\n dns_record_type = \"A\"",
}

// Omitted alerts default off on creation, explicit values reach PATCH, and
// omitting the attribute afterwards preserves the API value without a diff.
func TestDomainExpiryAlertsLifecycle(t *testing.T) {
	for resourceType, target := range domainExpiryTargets {
		t.Run(resourceType, func(t *testing.T) {
			api := newDomainExpiryAPI(t)
			defer api.server.Close()
			address := "onlineornot_" + resourceType + ".test"
			config := func(setting string) string {
				return domainExpiryConfig(api.server.URL, resourceType, target, setting)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{
					Config: config(""),
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue(address, tfjsonpath.New("domain_expiry_alerts"), knownvalue.Bool(false)),
					}},
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(address, "domain_expiry_alerts", "false"),
						api.expectLastBody("POST", false),
					),
				},
				{
					Config: config("domain_expiry_alerts = true"),
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(address, tfjsonpath.New("domain_expiry_alerts"), knownvalue.Bool(true)),
					}},
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(address, "domain_expiry_alerts", "true"),
						api.expectLastBody("PATCH", true),
					),
				},
				{Config: config(""), PlanOnly: true},
				{
					Config: config("domain_expiry_alerts = false"),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(address, "domain_expiry_alerts", "false"),
						api.expectLastBody("PATCH", false),
					),
				},
			}})
		})
	}
}

// Alerts enabled in the dashboard are read back and kept while omitted.
func TestDomainExpiryAlertsEnabledOutsideTerraform(t *testing.T) {
	api := newDomainExpiryAPI(t)
	defer api.server.Close()
	address := "onlineornot_tcp_check.test"
	config := domainExpiryConfig(api.server.URL, "tcp_check", domainExpiryTargets["tcp_check"], "")
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config},
		{
			PreConfig: func() {
				api.mu.Lock()
				defer api.mu.Unlock()
				api.checks["tcp-fixture"]["domain_expiry_alerts"] = true
			},
			Config:   config,
			PlanOnly: true,
		},
		{Config: config, Check: resource.TestCheckResourceAttr(address, "domain_expiry_alerts", "true")},
	}})
}

// The API's validation error reaches the user on create and on update.
func TestDomainExpiryAlertsRejectedTargets(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		api := newDomainExpiryAPI(t)
		defer api.server.Close()
		resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
			Config:      domainExpiryConfig(api.server.URL, "uptime_check", `url = "https://203.0.113.10/health"`, "domain_expiry_alerts = true"),
			ExpectError: regexp.MustCompile(`need\s+a\s+check\s+target\s+with\s+a\s+registered\s+domain`),
		}}})
	})
	t.Run("update", func(t *testing.T) {
		api := newDomainExpiryAPI(t)
		defer api.server.Close()
		target := "tcp_hostname = \"myapp.herokuapp.com\"\n tcp_port = 443"
		resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
			{Config: domainExpiryConfig(api.server.URL, "tcp_check", target, "")},
			{
				Config:      domainExpiryConfig(api.server.URL, "tcp_check", target, "domain_expiry_alerts = true"),
				ExpectError: regexp.MustCompile(`need\s+a\s+check\s+target\s+with\s+a\s+registered\s+domain`),
			},
		}})
	})
}

func TestDomainExpiryAlertsRejectScriptedBrowserChecks(t *testing.T) {
	for _, resourceType := range []string{"browser_check", "check"} {
		t.Run(resourceType, func(t *testing.T) {
			typeSetting := ""
			if resourceType == "check" {
				typeSetting = `type = "BROWSER_CHECK"`
			}
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
				Config:   domainExpiryConfig("http://127.0.0.1:1", resourceType, typeSetting+"\n script = \"test('home', async () => {})\"", "domain_expiry_alerts = true"),
				PlanOnly: true, ExpectError: regexp.MustCompile(`Domain\s+expiry\s+alerts\s+are\s+incompatible\s+with\s+scripted\s+browser\s+checks`),
			}}})
		})
	}
}
