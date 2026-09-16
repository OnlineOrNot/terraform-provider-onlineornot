package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

const headerSecret = "sentinel-secret-must-not-be-persisted"

type secretFreePlan struct{}

func (secretFreePlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	raw, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = err
		return
	}
	if strings.Contains(string(raw), headerSecret) {
		resp.Error = fmt.Errorf("secret found in plan")
	}
}

func TestWriteOnlyHeadersTerraformLifecycle(t *testing.T) {
	for _, kind := range []string{"uptime", "browser", ""} {
		name := kind + "_check"
		if kind == "" {
			name = "check"
		}
		t.Run(name, func(t *testing.T) {
			var mu sync.Mutex
			stored := map[string]any{}
			writes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "POST" || r.Method == "PATCH" {
					var input map[string]any
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						t.Error(err)
					}
					if _, ok := input["headers"]; ok {
						writes++
					}
					for k, v := range input {
						stored[k] = v
					}
					stored["id"] = "fixture123"
					stored["check_type"] = "UPTIME"
					if kind == "browser" {
						stored["check_type"] = "BROWSER"
					}
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
			}))
			defer server.Close()
			t.Setenv("TF_VAR_secret", headerSecret)
			config := func(label, fields string) string {
				return fmt.Sprintf(`
terraform { required_version = ">= 1.11.0" }
variable "secret" {
 type = string
 sensitive = true
 ephemeral = true
}
provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
resource "onlineornot_%s" "test" {
 name = %q
 url = "https://example.com"
 %s
}`, server.URL, name, label, fields)
			}
			wo := func(version int, value string) string {
				return fmt.Sprintf("write_only_headers = %s\nwrite_only_headers_version = %d", value, version)
			}
			address := "onlineornot_" + name + ".test"
			check := func(wantWrites int, remote string, ordinary bool) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					raw, err := json.Marshal(state)
					if err != nil {
						return err
					}
					if strings.Contains(string(raw), headerSecret) {
						return fmt.Errorf("secret found in persisted state")
					}
					attrs := state.RootModule().Resources[address].Primary.Attributes
					if attrs["write_only_headers.%"] != "" {
						return fmt.Errorf("write-only attribute persisted")
					}
					if !ordinary && attrs["headers.%"] != "" {
						return fmt.Errorf("secret headers populated ordinary state")
					}
					mu.Lock()
					defer mu.Unlock()
					if writes != wantWrites {
						return fmt.Errorf("header writes: got %d want %d", writes, wantWrites)
					}
					headers, _ := stored["headers"].(map[string]any)
					if remote == "" && len(headers) != 0 {
						return fmt.Errorf("headers not cleared")
					}
					if remote != "" && headers["Authorization"] != remote {
						return fmt.Errorf("remote headers did not match expected rotation")
					}
					return nil
				}
			}
			step := func(label, fields string, writes int, remote string, ordinary bool) resource.TestStep {
				return resource.TestStep{Config: config(label, fields), Check: check(writes, remote, ordinary), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{secretFreePlan{}}}}
			}
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				step("initial", wo(1, "{Authorization = var.secret}"), 1, headerSecret, false),
				{RefreshState: true, Check: check(1, headerSecret, false)},
				{ResourceName: address, ImportState: true, ImportStateCheck: func(states []*terraform.InstanceState) error {
					raw, err := json.Marshal(states)
					if err != nil {
						return err
					}
					if strings.Contains(string(raw), headerSecret) || states[0].Attributes["headers.%"] != "" {
						return fmt.Errorf("import leaked headers")
					}
					return nil
				}},
				step("unrelated", wo(1, "{Authorization = \"ignored-until-version-changes\"}"), 1, headerSecret, false),
				step("rotated", wo(2, "{Authorization = \"rotated\"}"), 2, "rotated", false),
				step("cleared", wo(3, "{}"), 3, "", false),
				step("secret-again", wo(4, "{Authorization = var.secret}"), 4, headerSecret, false),
				step("removed", "", 5, "", false),
				step("normal", "headers = {Authorization = \"public\"}", 6, "public", true),
				step("migration", wo(1, "{Authorization = var.secret}"), 7, headerSecret, false),
				step("normal-again", "headers = {Authorization = \"public\"}", 8, "public", true),
			}})
		})
	}
}

func TestWriteOnlyHeadersValidation(t *testing.T) {
	for _, fields := range []string{
		`write_only_headers = {Authorization = "secret"}`,
		`write_only_headers_version = 1`,
		"write_only_headers = {}\nwrite_only_headers_version = 0",
		"write_only_headers = {}\nwrite_only_headers_version = 1\nheaders = {}",
	} {
		resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
			Config:      "provider \"onlineornot\" { api_key = \"loopback-only\" }\nresource \"onlineornot_uptime_check\" \"test\" {\nname = \"test\"\nurl = \"https://example.com\"\n" + fields + "\n}",
			ExpectError: regexp.MustCompile("Write-only headers require a version|Invalid header version|Conflicting header modes"),
		}}})
	}
}

func TestWriteOnlyHeadersImportedMonitorUnrelatedUpdate(t *testing.T) {
	var mu sync.Mutex
	stored := map[string]any{"id": "fixture123", "name": "imported", "url": "https://example.com", "headers": map[string]string{"Authorization": headerSecret}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == "POST" {
			t.Error("import must not create")
		}
		if r.Method == "PATCH" {
			var input map[string]any
			json.NewDecoder(r.Body).Decode(&input)
			if _, ok := input["headers"]; ok {
				t.Error("unrelated imported update must preserve headers")
			}
			for k, v := range input {
				stored[k] = v
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": stored})
	}))
	defer server.Close()
	config := func(name string) string {
		return fmt.Sprintf(`
provider "onlineornot" {
 api_key = "loopback-only"
 base_url = %q
}
import {
 to = onlineornot_uptime_check.test
 id = "fixture123"
}
resource "onlineornot_uptime_check" "test" {
 name = %q
 url = "https://example.com"
}`, server.URL, name)
	}
	check := func(state *terraform.State) error {
		raw, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if strings.Contains(string(raw), headerSecret) {
			return fmt.Errorf("imported secret leaked into state")
		}
		return resource.TestCheckNoResourceAttr("onlineornot_uptime_check.test", "headers.%")(state)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config("imported"), Check: check, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{secretFreePlan{}}}},
		{Config: config("renamed"), Check: check, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{secretFreePlan{}}}},
		{RefreshState: true, Check: check},
	}})
}

// A server can apply a mutation and still return an error (or fail the next
// GET). Neither diagnostics nor a subsequent refresh may disclose its headers.
func TestWriteOnlyHeadersFailedMigration(t *testing.T) {
	for _, failMethod := range []string{"PATCH", "GET"} {
		t.Run(failMethod, func(t *testing.T) {
			ctx := context.Background()
			fail := true
			remote := &client.Check{ID: "fixture123", Name: "test", URL: "https://example.com", Headers: map[string]string{"Authorization": headerSecret}}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if fail && req.Method == failMethod {
					json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []map[string]any{{"message": headerSecret}}})
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "result": remote})
			}))
			defer server.Close()
			r := &CheckResource{client: client.NewClient(&client.Config{BaseURL: server.URL, APIKey: "loopback-only"})}
			var schema frameworkresource.SchemaResponse
			r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
			var model checkModel
			var diags diag.Diagnostics
			r.populateModelFromAPI(ctx, &model, remote, &diags)
			model.Headers, _ = types.MapValueFrom(ctx, types.StringType, map[string]string{"Accept": "application/json"})
			state := tfsdk.State{Schema: schema.Schema}
			diags.Append(state.Set(ctx, &model)...)
			model.Headers = types.MapNull(types.StringType)
			model.WriteOnlyHeadersVersion = types.Int64Value(1)
			plan := tfsdk.Plan{Schema: schema.Schema}
			diags.Append(plan.Set(ctx, &model)...)
			model.WriteOnlyHeaders, _ = types.MapValueFrom(ctx, types.StringType, remote.Headers)
			config := tfsdk.State{Schema: schema.Schema}
			diags.Append(config.Set(ctx, &model)...)
			if diags.HasError() {
				t.Fatal(diags)
			}
			update := frameworkresource.UpdateResponse{State: state}
			r.Update(ctx, frameworkresource.UpdateRequest{Plan: plan, State: state, Config: tfsdk.Config{Schema: schema.Schema, Raw: config.Raw}}, &update)
			if !update.Diagnostics.HasError() {
				t.Fatal("expected failure")
			}
			if strings.Contains(fmt.Sprint(update.Diagnostics), headerSecret) {
				t.Fatal("diagnostic leaked secret")
			}
			fail = false
			read := frameworkresource.ReadResponse{State: update.State}
			r.Read(ctx, frameworkresource.ReadRequest{State: update.State}, &read)
			if read.Diagnostics.HasError() {
				t.Fatal(read.Diagnostics)
			}
			if d := read.State.Get(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			if !model.Headers.IsNull() || !model.WriteOnlyHeaders.IsNull() {
				t.Fatal("failed migration leaked secret on refresh")
			}
			if !model.WriteOnlyHeadersVersion.IsNull() {
				t.Fatal("failed migration must retain prior version for retry")
			}
		})
	}
}
