package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

// UnitTest runs real Terraform plan/apply/import/refresh against a loopback API,
// without TF_ACC, credentials, or production traffic.
func TestTokenResourceLifecycle(t *testing.T) {
	var mu sync.Mutex
	tokens := map[string]client.Token{}
	created, deleted := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer mock-provider-key" {
			t.Error("unexpected credentials")
		}
		w.Header().Set("Content-Type", "application/json")
		id := strings.TrimPrefix(r.URL.Path, "/v1/tokens/")
		switch r.Method {
		case "POST":
			if r.URL.Path != "/v1/tokens" {
				t.Error(r.URL.Path)
			}
			var input client.CreateTokenRequest
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &input); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			created++
			id = fmt.Sprintf("token%d", created)
			defaultExpiry := "2029-01-01T00:00:00.000Z"
			expiry := &defaultExpiry
			if input.ExpiresAt != nil {
				expiry = *input.ExpiresAt
			} else if strings.Contains(string(body), `"expiresAt":null`) {
				expiry = nil
			}
			if expiry != nil {
				parsed, err := time.Parse(time.RFC3339Nano, *expiry)
				if err != nil {
					t.Error(err)
				}
				value := parsed.Format("2006-01-02T15:04:05.000Z")
				expiry = &value
			}
			token := client.Token{ID: id, Name: input.Name, ExpiresAfter: expiry, Grants: input.Grants}
			tokens[id] = token
			token.Grants = nil // The create contract does not return grants.
			token.Token = "mock-secret-" + id
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(client.APIResponse[client.Token]{Success: true, Result: token})
		case "GET":
			token, ok := tokens[id]
			if !ok {
				w.WriteHeader(404)
				fmt.Fprint(w, `{"success":false,"errors":[{"code":10006,"message":"gone"}]}`)
				return
			}
			// Server grant ordering is unrelated to HCL order.
			for a, b := 0, len(token.Grants)-1; a < b; a, b = a+1, b-1 {
				token.Grants[a], token.Grants[b] = token.Grants[b], token.Grants[a]
			}
			json.NewEncoder(w).Encode(client.APIResponse[client.Token]{Success: true, Result: token})
		case "DELETE":
			deleted++
			delete(tokens, id)
			// Simulate already-deleted token: Terraform destruction must still succeed.
			w.WriteHeader(404)
			fmt.Fprint(w, `{"success":false,"errors":[{"code":10006,"message":"gone"}]}`)
		default:
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(405)
		}
	}))
	defer server.Close()
	config := func(name, permission, expiry string) string {
		return fmt.Sprintf(`provider "onlineornot" {
 api_key = "mock-provider-key"
 base_url = %q
 }
resource "onlineornot_token" "test" {
 name = %q
 grants = [{scope="UPTIME_CHECKS",permission=%q},{scope="STATUS_PAGES",permission="READ"}]
 %s
}`, server.URL, name, permission, expiry)
	}
	check := func(id string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("onlineornot_token.test", "id", id),
			resource.TestCheckResourceAttr("onlineornot_token.test", "token", "mock-secret-"+id),
			resource.TestCheckResourceAttr("onlineornot_token.test", "grants.#", "2"),
		)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("first", "READ", `expires_at = "2030-01-01T00:00:00Z"`), Check: resource.ComposeAggregateTestCheckFunc(check("token1"), resource.TestCheckResourceAttr("onlineornot_token.test", "expires_at", "2030-01-01T00:00:00Z"))},
			{RefreshState: true, Check: check("token1")},
			{ResourceName: "onlineornot_token.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"token", "expires_at"}},
			{Config: config("renamed", "READ", `expires_at = "2030-01-01T00:00:00Z"`), Check: check("token2")},
			{Config: config("renamed", "EDIT", `expires_at = "2030-01-01T00:00:00Z"`), Check: check("token3")},
			{Config: config("renamed", "EDIT", `expires_at = "2031-01-01T00:00:00Z"`), Check: check("token4")},
			{Config: config("renamed", "EDIT", "never_expires = true"), Check: resource.ComposeAggregateTestCheckFunc(check("token5"), resource.TestCheckNoResourceAttr("onlineornot_token.test", "expires_at"))},
			{Config: config("renamed", "EDIT", "never_expires = true"), PreConfig: func() { mu.Lock(); delete(tokens, "token5"); mu.Unlock() }, Check: check("token6")},
			{Config: config("default", "READ", ""), Check: resource.ComposeAggregateTestCheckFunc(check("token7"), resource.TestCheckResourceAttr("onlineornot_token.test", "expires_after", "2029-01-01T00:00:00.000Z"))},
			{RefreshState: true, Check: check("token7")},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if created != 7 || deleted != 6 || len(tokens) != 0 {
		t.Errorf("created=%d deleted=%d remaining=%d", created, deleted, len(tokens))
	}
}

func TestTokenResourceValidationAndImport(t *testing.T) {
	ctx := context.Background()
	r := &TokenResource{}
	var schemaResp frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	if !s.Attributes["token"].IsSensitive() || !s.Attributes["token"].IsComputed() || s.Attributes["token"].IsOptional() {
		t.Fatal("secret must be sensitive computed only")
	}
	for _, id := range []string{"", "id/other", "id?x=1", "../verify", "verify", "permissions", "id#x", "id%2fother"} {
		resp := frameworkresource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
		r.ImportState(ctx, frameworkresource.ImportStateRequest{ID: id}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("accepted invalid import %q", id)
		}
	}
	for _, value := range []string{"not-a-date", "2030-01-01T00:00:00+01:00", "2030-01-01T00:00:00.000001Z"} {
		resp := validator.StringResponse{}
		tokenExpiryValidator{}.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue(value), Path: path.Root("expires_at")}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("accepted expiry %q", value)
		}
	}
	// Import cannot recover the secret, even after a refresh.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"result":{"id":"import123","name":"imported","expiresAfter":null,"grants":[]}}`)
	}))
	defer server.Close()
	r.client = client.NewClient(&client.Config{BaseURL: server.URL, APIKey: "mock"})
	imported := frameworkresource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.ImportState(ctx, frameworkresource.ImportStateRequest{ID: "import123"}, &imported)
	refreshed := frameworkresource.ReadResponse{State: imported.State}
	r.Read(ctx, frameworkresource.ReadRequest{State: imported.State}, &refreshed)
	var data tokenModel
	diags := refreshed.State.Get(ctx, &data)
	if imported.Diagnostics.HasError() || refreshed.Diagnostics.HasError() || diags.HasError() {
		t.Fatalf("import/read diagnostics: %v %v %v", imported.Diagnostics, refreshed.Diagnostics, diags)
	}
	if !data.Token.IsNull() || data.NeverExpires.ValueBool() != true || !data.ExpiresAfter.IsNull() {
		t.Fatal("unexpected imported state")
	}
}

func TestTokenResourceReadAndDeleteErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{401, 403, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"success":false,"errors":[{"code":10006,"message":"mock-secret 404 not found"}]}`)
			}))
			defer server.Close()
			r := &TokenResource{client: client.NewClient(&client.Config{BaseURL: server.URL, APIKey: "mock"})}
			var schemaResp frameworkresource.SchemaResponse
			r.Schema(ctx, frameworkresource.SchemaRequest{}, &schemaResp)
			state := tfsdk.State{Schema: schemaResp.Schema}
			data := tokenModel{ID: types.StringValue("id123"), Name: types.StringValue("name"), Token: types.StringValue("mock-secret"), Grants: types.SetNull(tokenGrantType), ExpiresAt: types.StringNull(), ExpiresAfter: types.StringNull(), NeverExpires: types.BoolValue(true)}
			if d := state.Set(ctx, &data); d.HasError() {
				t.Fatal(d)
			}
			read := frameworkresource.ReadResponse{State: state}
			r.Read(ctx, frameworkresource.ReadRequest{State: state}, &read)
			if read.Diagnostics.HasError() != (status != 404) || read.State.Raw.IsNull() != (status == 404) {
				t.Fatalf("unexpected read: %v", read.Diagnostics)
			}
			del := frameworkresource.DeleteResponse{State: state}
			r.Delete(ctx, frameworkresource.DeleteRequest{State: state}, &del)
			if del.Diagnostics.HasError() != (status != 404) {
				t.Fatalf("unexpected delete: %v", del.Diagnostics)
			}
			create := frameworkresource.CreateResponse{State: state}
			r.Create(ctx, frameworkresource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}}, &create)
			if !create.Diagnostics.HasError() {
				t.Fatal("expected create error")
			}
			for _, diags := range []diag.Diagnostics{read.Diagnostics, del.Diagnostics, create.Diagnostics} {
				if strings.Contains(fmt.Sprint(diags), "mock-secret") {
					t.Fatal("diagnostic leaked a secret")
				}
			}
		})
	}
}
