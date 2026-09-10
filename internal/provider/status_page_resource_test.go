package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_status_page"
)

func TestStatusPageResourceCreateHideFromSearchEngines(t *testing.T) {
	for _, tc := range []struct {
		name     string
		planned  types.Bool
		response string
		want     bool
	}{
		{"omitted_defaults_false", types.BoolUnknown(), ``, false},
		{"omitted_api_false", types.BoolUnknown(), `,"hide_from_search_engines":false`, false},
		{"omitted_api_true", types.BoolUnknown(), `,"hide_from_search_engines":true`, true},
		{"configured_false", types.BoolValue(false), ``, false},
		{"configured_true", types.BoolValue(true), ``, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPost || (req.URL.Path != "/v1/status_pages" && req.URL.Path != "/v1/status_pages/page1234") {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var sent client.StatusPage
				if err := json.NewDecoder(req.Body).Decode(&sent); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if sent.Name != "echo-test" || sent.Subdomain != "echo-test" || sent.HideFromSearchEngines != tc.planned.ValueBool() {
					t.Errorf("unexpected create payload: %+v", sent)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"success":true,"result":{"id":"page1234","name":"echo-test","subdomain":"echo-test"` + tc.response + `}}`))
			}))
			defer server.Close()

			r := &StatusPageResource{client: client.NewClient(&client.Config{BaseURL: server.URL, APIKey: "test-key"})}
			data := resource_status_page.StatusPageModel{
				Id:                    types.StringUnknown(),
				Name:                  types.StringValue("echo-test"),
				Subdomain:             types.StringValue("echo-test"),
				AllowedIps:            types.ListUnknown(types.StringType),
				CustomDomain:          types.StringUnknown(),
				Description:           types.StringUnknown(),
				Password:              types.StringUnknown(),
				HideFromSearchEngines: tc.planned,
			}
			plan := tfsdk.Plan{Schema: resource_status_page.StatusPageResourceSchema(ctx)}
			if diags := plan.Set(ctx, &data); diags.HasError() {
				t.Fatal(diags)
			}
			resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.State.Raw.IsFullyKnown() {
				t.Error("create returned unknown values after apply")
			}
			var got resource_status_page.StatusPageModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatal(diags)
			}
			if !got.HideFromSearchEngines.Equal(types.BoolValue(tc.want)) {
				t.Errorf("hide_from_search_engines = %s, want %t", got.HideFromSearchEngines, tc.want)
			}
			if got.Id.ValueString() != "page1234" || !got.Name.Equal(data.Name) || !got.Subdomain.Equal(data.Subdomain) {
				t.Errorf("unexpected status page identity: %+v", got)
			}
		})
	}
}

func TestStatusPageCreateSettingsFailureRetainsID(t *testing.T) {
	ctx := context.Background()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		if req.Method != "POST" {
			t.Errorf("unexpected method %s", req.Method)
		}
		if calls == 1 {
			w.Write([]byte(`{"success":true,"result":{"id":"page1234","name":"test","subdomain":"test"}}`))
			return
		}
		if req.URL.Path != "/v1/status_pages/page1234" {
			t.Errorf("unexpected settings URL: %s", req.URL.Path)
		}
		var input map[string]any
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if v, ok := input["description"]; !ok || v != "" {
			t.Errorf("empty description omitted: %v", input)
		}
		if v, ok := input["hide_from_search_engines"]; !ok || v != false {
			t.Errorf("false omitted: %v", input)
		}
		if ips, ok := input["allowed_ips"].([]any); !ok || len(ips) != 0 {
			t.Errorf("empty IPs omitted: %v", input)
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"errors":[{"message":"settings failed"}]}`))
	}))
	defer server.Close()
	r := &StatusPageResource{client: client.NewClient(&client.Config{BaseURL: server.URL, APIKey: "test"})}
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	data := resource_status_page.StatusPageModel{
		Id: types.StringUnknown(), Name: types.StringValue("test"), Subdomain: types.StringValue("test"),
		Description: types.StringValue(""), CustomDomain: types.StringNull(), Password: types.StringNull(),
		AllowedIps: types.ListValueMust(types.StringType, []attr.Value{}), HideFromSearchEngines: types.BoolValue(false),
	}
	if d := plan.Set(ctx, &data); d.HasError() {
		t.Fatal(d)
	}
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected settings failure")
	}
	if calls != 2 {
		t.Fatalf("expected create then update, got %d requests", calls)
	}
	var got resource_status_page.StatusPageModel
	if d := resp.State.Get(ctx, &got); d.HasError() {
		t.Fatal(d)
	}
	if got.Id.ValueString() != "page1234" || !resp.State.Raw.IsFullyKnown() {
		t.Fatalf("lost created state: %+v", got)
	}
}
