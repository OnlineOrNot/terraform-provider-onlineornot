package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// These tests use the same local API harness as inline scripts. Terraform itself
// reads the files, refreshes state, and plans resource address changes.
func TestScriptFileTerraformLifecycle(t *testing.T) {
	for _, resourceType := range []string{"browser_check", "check"} {
		t.Run(resourceType, func(t *testing.T) {
			kind := "browser"
			if resourceType == "check" {
				kind = ""
			}
			server, drift := mockCheckAPI(t, kind)
			defer server.Close()
			filename := filepath.Join(t.TempDir(), "homepage.spec.js")
			writeScript := func(script string) {
				t.Helper()
				if err := os.WriteFile(filename, []byte(script), 0600); err != nil {
					t.Fatal(err)
				}
			}
			writeScript(browserScript)
			address := "onlineornot_" + resourceType + ".test"
			config := fmt.Sprintf(`provider "onlineornot" {
 api_key = "fixture-key"
 base_url = %q
}
resource "onlineornot_%s" "test" {
 name = "file fixture"
 type = "BROWSER_CHECK"
 script = file(%q)
}`, server.URL, resourceType, filename)
			check := func(script string) resource.TestCheckFunc {
				return resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "id", "fixture"),
					resource.TestCheckResourceAttr(address, "script", script),
					resource.TestCheckNoResourceAttr(address, "timeout"),
				)
			}
			updated := strings.ReplaceAll(browserScript, "Example/", "Example Domain/")
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: config, Check: check(browserScript)},
					{Config: config, PlanOnly: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
					{PreConfig: func() { writeScript(updated) }, Config: config, Check: check(updated), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)}}},
					{Config: config, PlanOnly: true},
					{PreConfig: func() { drift(browserScript) }, Config: config, PlanOnly: true, ExpectNonEmptyPlan: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)}}},
					{Config: config, Check: check(updated)},
					{Config: config, PlanOnly: true},
				},
			})
		})
	}
}

func TestScriptFilesetTerraformLifecycle(t *testing.T) {
	server, _ := mockCheckAPI(t, "browser")
	defer server.Close()
	dir := t.TempDir()
	second := strings.ReplaceAll(browserScript, "Example/", "Example Domain/")
	for filename, script := range map[string]string{"first.spec.js": browserScript, "second.spec.js": second, "helper.js": "not a test"} {
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(script), 0600); err != nil {
			t.Fatal(err)
		}
	}
	config := fmt.Sprintf(`provider "onlineornot" {
 api_key = "fixture-key"
 base_url = %q
}
resource "onlineornot_browser_check" "files" {
 for_each = fileset(%q, "*.spec.js")
 name = each.key
 script = file("%s/${each.key}")
}`, server.URL, dir, filepath.ToSlash(dir))
	first := `onlineornot_browser_check.files["first.spec.js"]`
	secondAddress := `onlineornot_browser_check.files["second.spec.js"]`
	renamed := `onlineornot_browser_check.files["renamed.spec.js"]`
	// plugin-testing's legacy state shim cannot represent for_each string keys.
	// Use the Terraform CLI only for this case, with the same loopback API and
	// a locally built provider. Never use a provider token or remote backend.
	terraformPath := os.Getenv("TF_ACC_TERRAFORM_PATH")
	if terraformPath == "" {
		var err error
		terraformPath, err = exec.LookPath("terraform")
		if err != nil {
			t.Skip("Terraform CLI required; set TF_ACC_TERRAFORM_PATH")
		}
	}
	providerDir := t.TempDir()
	build := exec.Command("go", "build", "-o", filepath.Join(providerDir, "terraform-provider-onlineornot"), "../..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build provider: %v\n%s", err, output)
	}
	cliConfig := filepath.Join(t.TempDir(), "terraform.rc")
	if err := os.WriteFile(cliConfig, []byte(fmt.Sprintf(`provider_installation {
 dev_overrides { "registry.terraform.io/onlineornot/onlineornot" = %q }
 direct {}
}`, providerDir)), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) []byte {
		t.Helper()
		command := exec.Command(terraformPath, args...)
		command.Dir = dir
		command.Env = append(os.Environ(), "TF_CLI_CONFIG_FILE="+cliConfig, "TF_INPUT=0", "TF_IN_AUTOMATION=1", "ONLINEORNOT_API_KEY=", "TF_CLI_ARGS=", "TF_CLI_ARGS_apply=", "TF_CLI_ARGS_plan=", "TF_CLI_ARGS_show=")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("terraform %v: %v\n%s", args, err, output)
		}
		return output
	}
	config = `terraform {
 required_providers {
  onlineornot = { source = "onlineornot/onlineornot" }
 }
}
` + config
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	run("apply", "-auto-approve", "-no-color")
	var state struct {
		Values struct {
			RootModule struct {
				Resources []struct {
					Address string
					Values  struct {
						ID     string
						Script string
					}
				}
			} `json:"root_module"`
		}
	}
	if err := json.Unmarshal(run("show", "-json"), &state); err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{}
	ids := map[string]bool{}
	for _, res := range state.Values.RootModule.Resources {
		sources[res.Address] = res.Values.Script
		ids[res.Values.ID] = true
	}
	if !reflect.DeepEqual(sources, map[string]string{first: browserScript, secondAddress: second}) || len(ids) != 2 || ids[""] {
		t.Fatalf("unexpected fileset state: %v, IDs: %v", sources, ids)
	}
	planActions := func() map[string][]string {
		run("plan", "-out=plan.tfplan", "-no-color")
		var plan struct {
			ResourceChanges []struct {
				Address string
				Change  struct{ Actions []string }
			} `json:"resource_changes"`
		}
		if err := json.Unmarshal(run("show", "-json", "plan.tfplan"), &plan); err != nil {
			t.Fatal(err)
		}
		actions := map[string][]string{}
		for _, res := range plan.ResourceChanges {
			actions[res.Address] = res.Change.Actions
		}
		return actions
	}
	if actions := planActions(); !reflect.DeepEqual(actions, map[string][]string{first: {"no-op"}, secondAddress: {"no-op"}}) {
		t.Fatalf("non-empty plan: %v", actions)
	}
	if err := os.Rename(filepath.Join(dir, "first.spec.js"), filepath.Join(dir, "renamed.spec.js")); err != nil {
		t.Fatal(err)
	}
	if actions := planActions(); !reflect.DeepEqual(actions, map[string][]string{first: {"delete"}, renamed: {"create"}, secondAddress: {"no-op"}}) {
		t.Fatalf("unexpected rename actions: %v", actions)
	}
}
