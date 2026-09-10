package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
				if req.Method != http.MethodPost || req.URL.Path != "/v1/status_pages" {
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
