package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

func TestEnvironmentVariableSchema(t *testing.T) {
	var resp frameworkresource.SchemaResponse
	(&EnvironmentVariableResource{}).Schema(context.Background(), frameworkresource.SchemaRequest{}, &resp)
	value := resp.Schema.Attributes["value"]
	if !value.IsWriteOnly() || !value.IsSensitive() || value.IsComputed() {
		t.Fatal("secret value must be sensitive write-only")
	}
	if resp.Schema.Attributes["value_version"].IsSensitive() {
		t.Fatal("change trigger must not be a secret")
	}
}

// Terraform CLI integration against a metadata-only, local API. No TF_ACC or live credentials.
func TestEnvironmentVariableResourceLifecycle(t *testing.T) {
	var mu sync.Mutex
	var remote *client.EnvironmentVariable
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer mock-key" {
			t.Error("missing auth")
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/env") {
			t.Error(r.URL.Path)
		}
		switch r.Method {
		case "POST", "PATCH":
			var input client.EnvironmentVariableWrite
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if r.Method == "POST" {
				if input.Type != "secret" || input.Value == nil {
					t.Error("create requires secret")
				}
				remote = &client.EnvironmentVariable{ID: "env123", Name: input.Name, Type: "secret"}
			} else {
				if input.Type != "" {
					t.Error("type must not be patched")
				}
				if input.Name != "" {
					remote.Name = input.Name
				}
			}
			if input.Value != nil {
				requests = append(requests, *input.Value)
			} else {
				requests = append(requests, "<omitted>")
			}
			json.NewEncoder(w).Encode(client.APIResponse[client.EnvironmentVariable]{Success: true, Result: *remote})
		case "GET":
			if remote == nil {
				w.WriteHeader(404)
				fmt.Fprint(w, `{"success":false}`)
				return
			}
			json.NewEncoder(w).Encode(client.APIResponse[client.EnvironmentVariable]{Success: true, Result: *remote})
		case "DELETE":
			remote = nil
			fmt.Fprint(w, `{"success":true,"result":{"id":"env123"}}`)
		default:
			t.Error(r.Method)
		}
	}))
	defer server.Close()
	config := func(name, value, version string) string {
		return fmt.Sprintf(`terraform { required_version = ">= 1.11" }
provider "onlineornot" {
 api_key = "mock-key"
 base_url = %q
}
resource "onlineornot_environment_variable" "test" {
 name = %q
 type = "secret"
 value = %q
 value_version = %q
}
output "env_test_header" {
 value = onlineornot_environment_variable.test.reference
}
output "authorization_header" {
 value = "Bearer ${onlineornot_environment_variable.test.reference}"
}`, server.URL, name, value, version)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config("API_TOKEN", "first-secret", "1"), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("onlineornot_environment_variable.test", "id", "env123"),
			resource.TestCheckResourceAttr("onlineornot_environment_variable.test", "name", "API_TOKEN"),
			resource.TestCheckResourceAttr("onlineornot_environment_variable.test", "reference", "{{API_TOKEN}}"),
			resource.TestCheckOutput("env_test_header", "{{API_TOKEN}}"),
			resource.TestCheckOutput("authorization_header", "Bearer {{API_TOKEN}}"),
			resource.TestCheckNoResourceAttr("onlineornot_environment_variable.test", "value"),
		)},
		{RefreshState: true, Check: resource.TestCheckResourceAttr("onlineornot_environment_variable.test", "reference", "{{API_TOKEN}}")},
		{Config: config("RENAMED_TOKEN", "first-secret", "1"), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectKnownValue("onlineornot_environment_variable.test", tfjsonpath.New("id"), knownvalue.StringExact("env123")),
			plancheck.ExpectKnownValue("onlineornot_environment_variable.test", tfjsonpath.New("reference"), knownvalue.StringExact("{{RENAMED_TOKEN}}")),
		}}, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("onlineornot_environment_variable.test", "name", "RENAMED_TOKEN"),
			resource.TestCheckResourceAttr("onlineornot_environment_variable.test", "reference", "{{RENAMED_TOKEN}}"),
			resource.TestCheckOutput("env_test_header", "{{RENAMED_TOKEN}}"),
			resource.TestCheckOutput("authorization_header", "Bearer {{RENAMED_TOKEN}}"),
			resource.TestCheckNoResourceAttr("onlineornot_environment_variable.test", "value"),
		)},
		{RefreshState: true, Check: resource.TestCheckResourceAttr("onlineornot_environment_variable.test", "reference", "{{RENAMED_TOKEN}}")},
		{ResourceName: "onlineornot_environment_variable.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"value_version"}},
		{Config: config("RENAMED_TOKEN", "second-secret", "2"), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectKnownValue("onlineornot_environment_variable.test", tfjsonpath.New("id"), knownvalue.StringExact("env123")),
			plancheck.ExpectKnownValue("onlineornot_environment_variable.test", tfjsonpath.New("reference"), knownvalue.StringExact("{{RENAMED_TOKEN}}")),
		}}, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("onlineornot_environment_variable.test", "value_version", "2"),
			resource.TestCheckResourceAttr("onlineornot_environment_variable.test", "reference", "{{RENAMED_TOKEN}}"),
		)},
	}})
	mu.Lock()
	defer mu.Unlock()
	if remote != nil || strings.Join(requests, ",") != "first-secret,<omitted>,second-secret" {
		t.Errorf("remote=%+v requests=%v", remote, requests)
	}
}
