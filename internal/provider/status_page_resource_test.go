package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
		want     types.Bool
	}{
		{"omitted_api_null", types.BoolUnknown(), `,"hide_from_search_engines":null`, types.BoolNull()},
		{"omitted_api_false", types.BoolUnknown(), `,"hide_from_search_engines":false`, types.BoolValue(false)},
		{"omitted_api_true", types.BoolUnknown(), `,"hide_from_search_engines":true`, types.BoolValue(true)},
		{"configured_false", types.BoolValue(false), ``, types.BoolValue(false)},
		{"configured_true", types.BoolValue(true), ``, types.BoolValue(true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodGet {
					w.Write([]byte(`{"success":true,"result":{"id":"page1234"` + tc.response + `}}`))
					return
				}
				if req.Method != http.MethodPost || (req.URL.Path != "/v1/status_pages" && req.URL.Path != "/v1/status_pages/page1234") {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var sent client.StatusPageInput
				if err := json.NewDecoder(req.Body).Decode(&sent); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if sent.Name != "echo-test" || sent.Subdomain != "echo-test" || (sent.HideFromSearchEngines != nil && *sent.HideFromSearchEngines != tc.planned.ValueBool()) {
					t.Errorf("unexpected create payload: %+v", sent)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"success":true,"result":{"id":"page1234","name":"echo-test","subdomain":"echo-test"}}`))
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
			if !got.HideFromSearchEngines.Equal(tc.want) {
				t.Errorf("hide_from_search_engines = %s, want %s", got.HideFromSearchEngines, tc.want)
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

func TestStatusPageImageInputs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   types.String
		prior   *resource_status_page.StatusPageModel
		present bool
		want    any
	}{
		{name: "unmanaged", value: types.StringNull()},
		{name: "unknown", value: types.StringUnknown()},
		{name: "upload", value: types.StringValue("data:image/png;base64,aGVsbG8="), present: true, want: "data:image/png;base64,aGVsbG8="},
		{name: "remove", value: types.StringNull(), prior: &resource_status_page.StatusPageModel{Logo: types.StringValue("old"), DarkLogo: types.StringValue("old"), Favicon: types.StringValue("old")}, present: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := resource_status_page.StatusPageModel{Logo: tc.value, DarkLogo: tc.value, Favicon: tc.value}
			input := statusPageInput(context.Background(), &data)
			applyStatusPageImages(input, &data, tc.prior)
			body, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"logo", "dark_logo", "favicon"} {
				got, present := payload[name]
				if present != tc.present || got != tc.want {
					t.Errorf("%s = %v (present %v), want %v (present %v)", name, got, present, tc.want, tc.present)
				}
			}
		})
	}
}

func TestStatusPageImagesAreOptionalInputs(t *testing.T) {
	var resp resource.SchemaResponse
	(&StatusPageResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	for _, name := range []string{"logo", "dark_logo", "favicon"} {
		a := resp.Schema.Attributes[name].(schema.StringAttribute)
		if !a.Optional || a.Computed {
			t.Errorf("%s must be optional, not computed", name)
		}
	}
}

func TestStatusPageImageLifecycle(t *testing.T) {
	ctx := context.Background()
	image := "data:image/png;base64,aGVsbG8="
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			writes++
			var payload map[string]any
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"logo", "dark_logo", "favicon"} {
				value, present := payload[name]
				if !present || (writes == 1 && value != image) || (writes == 2 && value != nil) {
					t.Errorf("unexpected image request: %v", payload)
				}
			}
		}
		w.Write([]byte(`{"success":true,"result":{"id":"page1234","name":"test","subdomain":"test","hide_from_search_engines":false,"logo_url":"https://example.com/logo.png"}}`))
	}))
	defer server.Close()
	r := &StatusPageResource{client: client.NewClient(&client.Config{BaseURL: server.URL, APIKey: "test"})}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	data := resource_status_page.StatusPageModel{
		Id: types.StringUnknown(), Name: types.StringValue("test"), Subdomain: types.StringValue("test"),
		Logo: types.StringValue(image), DarkLogo: types.StringValue(image), Favicon: types.StringValue(image),
		HideFromSearchEngines: types.BoolUnknown(), AllowedIps: types.ListNull(types.StringType),
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, &data); diags.HasError() {
		t.Fatal(diags)
	}
	created := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	refreshed := resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, &refreshed)
	if refreshed.Diagnostics.HasError() {
		t.Fatal(refreshed.Diagnostics)
	}
	if diags := refreshed.State.Get(ctx, &data); diags.HasError() {
		t.Fatal(diags)
	}
	if data.Logo.ValueString() != image || data.DarkLogo.ValueString() != image || data.Favicon.ValueString() != image {
		t.Fatal("refresh must preserve image inputs, not replace them with hosted URLs")
	}
	data.Logo, data.DarkLogo, data.Favicon = types.StringNull(), types.StringNull(), types.StringNull()
	if diags := plan.Set(ctx, &data); diags.HasError() {
		t.Fatal(diags)
	}
	updated := resource.UpdateResponse{State: refreshed.State}
	r.Update(ctx, resource.UpdateRequest{State: refreshed.State, Plan: plan}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}
	if diags := updated.State.Get(ctx, &data); diags.HasError() {
		t.Fatal(diags)
	}
	if !data.Logo.IsNull() || !data.DarkLogo.IsNull() || !data.Favicon.IsNull() {
		t.Fatal("removed inputs must remain null")
	}
	if writes != 2 {
		t.Errorf("expected create and update, got %d writes", writes)
	}
}
