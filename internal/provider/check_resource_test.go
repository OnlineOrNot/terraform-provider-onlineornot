package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_check"
)

func TestCheckModelToClientPreservesHeadersAndAssertions(t *testing.T) {
	ctx := context.Background()
	headers := map[string]string{
		"Authorization": "Bearer example",
		"Content-Type":  "application/json",
	}
	headerValues, headerDiags := types.MapValueFrom(ctx, types.StringType, headers)
	if headerDiags.HasError() {
		t.Fatalf("failed to create test headers: %v", headerDiags.Errors())
	}

	assertion := resource_check.NewAssertionsValueMust(
		resource_check.AssertionsValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"type":       types.StringValue("JSON_BODY"),
			"property":   types.StringValue("$.status"),
			"comparison": types.StringValue("EQUALS"),
			"expected":   types.StringValue("ok"),
		},
	)
	assertionType := resource_check.AssertionsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_check.AssertionsValue{}.AttributeTypes(ctx),
		},
	}
	assertionValues, assertionDiags := types.ListValueFrom(ctx, assertionType, []resource_check.AssertionsValue{assertion})
	if assertionDiags.HasError() {
		t.Fatalf("failed to create test assertions: %v", assertionDiags.Errors())
	}

	model := checkModel{
		Name:       types.StringValue("API check"),
		Url:        types.StringValue("https://example.com"),
		Headers:    headerValues,
		Assertions: assertionValues,
	}
	var diagnostics diag.Diagnostics

	check := checkModelToClient(ctx, &model, "UPTIME_CHECK", &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("unexpected conversion diagnostics: %v", diagnostics.Errors())
	}
	if !reflect.DeepEqual(check.Headers, headers) {
		t.Errorf("expected headers %v, got %v", headers, check.Headers)
	}
	expectedAssertions := []client.Assertion{{
		Type:       "JSON_BODY",
		Property:   "$.status",
		Comparison: "EQUALS",
		Expected:   "ok",
	}}
	if !reflect.DeepEqual(check.Assertions, expectedAssertions) {
		t.Errorf("expected assertions %v, got %v", expectedAssertions, check.Assertions)
	}
	if check.Type != "UPTIME_CHECK" {
		t.Errorf("expected forced type UPTIME_CHECK, got %s", check.Type)
	}
}

func TestCheckModelToClientPreservesBasicAuthValues(t *testing.T) {
	ctx := context.Background()
	model := checkModel{
		AuthUsername: types.StringValue(""),
		AuthPassword: types.StringValue("secret"),
	}
	var diagnostics diag.Diagnostics

	check := checkModelToClient(ctx, &model, "UPTIME_CHECK", &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("unexpected conversion diagnostics: %v", diagnostics.Errors())
	}
	if check.AuthUsername == nil || *check.AuthUsername != "" {
		t.Errorf("expected explicit empty username, got %#v", check.AuthUsername)
	}
	if check.AuthPassword == nil || *check.AuthPassword != "secret" {
		t.Errorf("expected password to be preserved, got %#v", check.AuthPassword)
	}
}

func TestCheckResourceSchemaMarksAuthPasswordSensitive(t *testing.T) {
	var response frameworkresource.SchemaResponse
	resource := CheckResource{}

	resource.Schema(context.Background(), frameworkresource.SchemaRequest{}, &response)

	authPassword, ok := response.Schema.Attributes["auth_password"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("expected auth_password to be a string attribute, got %T", response.Schema.Attributes["auth_password"])
	}
	if !authPassword.Sensitive {
		t.Error("expected auth_password to be sensitive")
	}
}

func TestCheckResourcePopulateModelFromAPIReconcilesBasicAuthState(t *testing.T) {
	empty := ""
	user := "user"
	secret := "secret"
	tests := []struct {
		name             string
		username         types.String
		password         types.String
		check            *client.Check
		expectedUsername types.String
		expectedPassword types.String
	}{
		{
			name:             "explicit empty values returned by API",
			username:         types.StringValue("stale-user"),
			password:         types.StringValue("stale-password"),
			check:            &client.Check{AuthUsername: &empty, AuthPassword: &empty},
			expectedUsername: types.StringValue(""),
			expectedPassword: types.StringValue(""),
		},
		{
			name:             "credentials removed outside Terraform",
			username:         types.StringValue("user"),
			password:         types.StringValue("secret"),
			check:            &client.Check{},
			expectedUsername: types.StringNull(),
			expectedPassword: types.StringNull(),
		},
		{
			name:             "computed values absent from API response",
			username:         types.StringUnknown(),
			password:         types.StringUnknown(),
			check:            &client.Check{},
			expectedUsername: types.StringNull(),
			expectedPassword: types.StringNull(),
		},
		{
			name:             "credentials returned by API",
			username:         types.StringNull(),
			password:         types.StringNull(),
			check:            &client.Check{AuthUsername: &user, AuthPassword: &secret},
			expectedUsername: types.StringValue("user"),
			expectedPassword: types.StringValue("secret"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := checkModel{
				AuthUsername: test.username,
				AuthPassword: test.password,
			}
			var diagnostics diag.Diagnostics
			resource := CheckResource{}

			resource.populateModelFromAPI(context.Background(), &model, test.check, &diagnostics)

			if diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diagnostics.Errors())
			}
			if !model.AuthUsername.Equal(test.expectedUsername) {
				t.Errorf("expected username %s, got %s", test.expectedUsername, model.AuthUsername)
			}
			if !model.AuthPassword.Equal(test.expectedPassword) {
				t.Errorf("expected password %s, got %s", test.expectedPassword, model.AuthPassword)
			}
		})
	}
}

func TestCheckModelToClientSkipsUnknownCollections(t *testing.T) {
	ctx := context.Background()
	unknownStrings := types.ListUnknown(types.StringType)
	assertionType := resource_check.AssertionsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_check.AssertionsValue{}.AttributeTypes(ctx),
		},
	}
	model := checkModel{
		Name:                 types.StringValue("API check"),
		Url:                  types.StringValue("https://example.com"),
		TestRegions:          unknownStrings,
		UserAlerts:           unknownStrings,
		SlackAlerts:          unknownStrings,
		DiscordAlerts:        unknownStrings,
		TelegramAlerts:       unknownStrings,
		PushoverAlerts:       unknownStrings,
		WebhookAlerts:        unknownStrings,
		OncallAlerts:         unknownStrings,
		IncidentIoAlerts:     unknownStrings,
		MicrosoftTeamsAlerts: unknownStrings,
		Headers:              types.MapUnknown(types.StringType),
		Assertions:           types.ListUnknown(assertionType),
	}
	var diagnostics diag.Diagnostics

	check := checkModelToClient(ctx, &model, "UPTIME_CHECK", &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics for unknown computed values: %v", diagnostics.Errors())
	}
	if check.TestRegions != nil || check.UserAlerts != nil || check.Headers != nil || check.Assertions != nil {
		t.Errorf("expected unknown collections to be omitted, got %#v", check)
	}
}

func TestCheckResourceUpdateWithoutOperationalStateChanges(t *testing.T) {
	ctx := context.Background()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCount++
		if req.Method != http.MethodPatch {
			t.Errorf("expected PATCH request, got %s", req.Method)
		}
		if req.URL.Path != "/v1/checks/check-id" {
			t.Errorf("expected check update path, got %s", req.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode update payload: %v", err)
		}
		if _, ok := payload["paused"]; ok {
			t.Error("expected ordinary update to omit paused")
		}
		if _, ok := payload["muted"]; ok {
			t.Error("expected ordinary update to omit muted")
		}

		if err := json.NewEncoder(w).Encode(client.APIResponse[client.Check]{
			Success: true,
			Result: client.Check{
				ID:     "check-id",
				Name:   "updated check",
				URL:    "https://example.org",
				Method: "GET",
			},
		}); err != nil {
			t.Fatalf("failed to encode update response: %v", err)
		}
	}))
	defer server.Close()

	checkResource := CheckResource{client: client.NewClient(&client.Config{BaseURL: server.URL})}
	var schemaResponse frameworkresource.SchemaResponse
	checkResource.Schema(ctx, frameworkresource.SchemaRequest{}, &schemaResponse)

	stateModel := checkModel{}
	var modelDiagnostics diag.Diagnostics
	checkResource.populateModelFromAPI(ctx, &stateModel, &client.Check{
		ID:     "check-id",
		Name:   "original check",
		URL:    "https://example.com",
		Method: "GET",
	}, &modelDiagnostics)
	if modelDiagnostics.HasError() {
		t.Fatalf("failed to create state model: %v", modelDiagnostics.Errors())
	}
	planModel := stateModel
	planModel.Name = types.StringValue("updated check")
	planModel.Url = types.StringValue("https://example.org")

	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diagnostics := plan.Set(ctx, &planModel); diagnostics.HasError() {
		t.Fatalf("failed to create update plan: %v", diagnostics.Errors())
	}
	state := tfsdk.State{Schema: schemaResponse.Schema}
	if diagnostics := state.Set(ctx, &stateModel); diagnostics.HasError() {
		t.Fatalf("failed to create prior state: %v", diagnostics.Errors())
	}
	response := frameworkresource.UpdateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}

	checkResource.Update(ctx, frameworkresource.UpdateRequest{Plan: plan, State: state}, &response)

	if response.Diagnostics.HasError() {
		t.Fatalf("unexpected update diagnostics: %v", response.Diagnostics.Errors())
	}
	if requestCount != 1 {
		t.Fatalf("expected one update request, got %d", requestCount)
	}
}

func TestAccCheckResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCheckResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onlineornot_check.test", "name", rName),
					resource.TestCheckResourceAttr("onlineornot_check.test", "url", "https://example.com"),
					resource.TestCheckResourceAttr("onlineornot_check.test", "method", "GET"),
					resource.TestCheckResourceAttrSet("onlineornot_check.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "onlineornot_check.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccCheckResourceConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onlineornot_check.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("onlineornot_check.test", "url", "https://example.org"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccCheckResource_withAlerts(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceConfig_withUserAlerts(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onlineornot_check.test", "name", rName),
					resource.TestCheckResourceAttrSet("onlineornot_check.test", "user_alerts.#"),
				),
			},
		},
	})
}

func testAccCheckResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "onlineornot_check" "test" {
  name = %[1]q
  url  = "https://example.com"
}
`, name)
}

func testAccCheckResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "onlineornot_check" "test" {
  name = "%[1]s-updated"
  url  = "https://example.org"
}
`, name)
}

func testAccCheckResourceConfig_withUserAlerts(name string) string {
	return fmt.Sprintf(`
data "onlineornot_users" "all" {}

resource "onlineornot_check" "test" {
  name        = %[1]q
  url         = "https://example.com"
  user_alerts = [data.onlineornot_users.all.users[0].id]
}
`, name)
}
