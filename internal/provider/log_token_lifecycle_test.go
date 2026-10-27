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
)

func TestLogTokenLifecycle(t *testing.T) {
	for _, permission := range []string{"READ", "EDIT"} {
		t.Run(permission, func(t *testing.T) {
			var mu sync.Mutex
			tokens := map[string]map[string]any{}
			projects := []any{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Header.Get("Authorization") != "Bearer loopback-only" {
					t.Error("unexpected provider credentials")
				}
				id := strings.TrimPrefix(r.URL.Path, "/v1/tokens/")
				switch r.Method {
				case "POST":
					var input map[string]any
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					if r.URL.Path != "/v1/tokens" {
						t.Error(r.URL.Path)
					}
					grants, ok := input["grants"].([]any)
					if !ok || len(grants) != 3 {
						t.Errorf("unexpected combined grants: %v", input["grants"])
					}
					projects = append(projects, input["project_id"])
					id = fmt.Sprintf("token%d", len(projects))
					result := map[string]any{"id": id, "name": input["name"], "grants": input["grants"], "project_id": input["project_id"], "expiresAfter": "2030-01-01T00:00:00.000Z"}
					tokens[id] = result
					created := map[string]any{}
					for key, value := range result {
						created[key] = value
					}
					delete(created, "grants")
					created["token"] = "test-secret-" + id
					w.WriteHeader(201)
					json.NewEncoder(w).Encode(map[string]any{"success": true, "result": created})
				case "GET":
					token, exists := tokens[id]
					if !exists {
						w.WriteHeader(404)
						return
					}
					json.NewEncoder(w).Encode(map[string]any{"success": true, "result": token})
				case "DELETE":
					delete(tokens, id)
					fmt.Fprint(w, `{"success":true,"result":{"deleted":true}}`)
				default:
					t.Errorf("unexpected method %s", r.Method)
					w.WriteHeader(405)
				}
			}))
			defer server.Close()
			config := func(project string) string {
				return fmt.Sprintf(`provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_token" "logs" {
 name = "Application logs"
 expires_at = "2030-01-01T00:00:00.000Z"
 grants = [{scope="LOGS",permission="SEND"},{scope="LOGS",permission=%q},{scope="API_TOKENS",permission="READ"}]
 %s
}`, server.URL, permission, project)
			}
			address := "onlineornot_token.logs"
			assigned := config(`project_id = "projectSource123456"`)
			check := func(id, project string) resource.TestCheckFunc {
				checks := []resource.TestCheckFunc{resource.TestCheckResourceAttr(address, "id", id), resource.TestCheckResourceAttr(address, "grants.#", "3")}
				if project == "" {
					checks = append(checks, resource.TestCheckNoResourceAttr(address, "project_id"))
				} else {
					checks = append(checks, resource.TestCheckResourceAttr(address, "project_id", project))
				}
				return resource.ComposeAggregateTestCheckFunc(checks...)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: assigned, Check: resource.ComposeTestCheckFunc(check("token1", "projectSource123456"), resource.TestCheckResourceAttr(address, "token", "test-secret-token1"))},
				{RefreshState: true, Check: resource.TestCheckResourceAttr(address, "token", "test-secret-token1")},
				{Config: assigned, ResourceName: address, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"token"}},
				{Config: assigned, PlanOnly: true, ExpectNonEmptyPlan: false},
				{RefreshState: true, Check: resource.ComposeTestCheckFunc(check("token1", "projectSource123456"), resource.TestCheckResourceAttr(address, "token", "test-secret-token1"))},
				{Config: config(`project_id = "projectTarget123456"`), Check: check("token2", "projectTarget123456")},
				{Config: config(""), Check: check("token3", "")},
				{Config: config("project_id = null"), PlanOnly: true, ExpectNonEmptyPlan: false},
				{Config: assigned, Check: check("token4", "projectSource123456")},
			}})
			mu.Lock()
			defer mu.Unlock()
			expected := []any{"projectSource123456", "projectTarget123456", nil, "projectSource123456"}
			if fmt.Sprint(projects) != fmt.Sprint(expected) || len(tokens) != 0 {
				t.Errorf("destinations=%v remaining=%v", projects, len(tokens))
			}
		})
	}
}

func TestLogTokenImportPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v1/tokens/importedlogs123" {
			t.Error(r.URL.Path)
		}
		switch r.Method {
		case "GET":
			fmt.Fprint(w, `{"success":true,"result":{"id":"importedlogs123","name":"Existing logs","project_id":"projectSource123456","expiresAfter":null,"grants":[{"scope":"LOGS","permission":"SEND"},{"scope":"API_TOKENS","permission":"READ"}]}}`)
		case "DELETE":
			fmt.Fprint(w, `{"success":true,"result":{"deleted":true}}`)
		default:
			t.Errorf("import/plan issued %s", r.Method)
			w.WriteHeader(405)
		}
	}))
	defer server.Close()
	config := fmt.Sprintf(`provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_token" "logs" {
 name = "Existing logs"
 project_id = "projectSource123456"
 grants = [{scope="LOGS",permission="SEND"},{scope="API_TOKENS",permission="READ"}]
 never_expires = true
}`, server.URL)
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, ResourceName: "onlineornot_token.logs", ImportState: true, ImportStateId: "importedlogs123", ImportStatePersist: true},
		{Config: config, PlanOnly: true, ExpectNonEmptyPlan: false},
		{RefreshState: true, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("onlineornot_token.logs", "project_id", "projectSource123456"), resource.TestCheckNoResourceAttr("onlineornot_token.logs", "token"))},
		{Config: config, PlanOnly: true, ExpectNonEmptyPlan: false},
	}})
}
