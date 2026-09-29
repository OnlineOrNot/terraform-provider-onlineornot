package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

func TestProjectSelectionSchemas(t *testing.T) {
	for name, r := range map[string]resource.Resource{"check": NewCheckResource(), "uptime": NewUptimeCheckResource(), "browser": NewBrowserCheckResource(), "dns": NewDNSCheckResource(), "tcp": NewTCPCheckResource(), "heartbeat": NewHeartbeatResource(), "variable": NewEnvironmentVariableResource()} {
		t.Run(name, func(t *testing.T) {
			var resp resource.SchemaResponse
			r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
			a, ok := resp.Schema.Attributes["project_id"].(schema.StringAttribute)
			if !ok || !a.Optional || !a.Computed {
				t.Fatal("project_id must be optional/computed for Default compatibility")
			}
			if len(a.Validators) == 0 {
				t.Fatal("missing encoded ID validation")
			}
		})
	}
}

func TestProjectMoveFailurePreservesState(t *testing.T) {
	for _, code := range []int{403, 404, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != "POST" || r.URL.Path != "/v1/checks/check12345678/move" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(code)
				w.Write([]byte(`{"success":false,"errors":[{"message":"sensitive response must not reach diagnostics"}]}`))
			}))
			defer server.Close()
			c := client.NewClient(&client.Config{APIKey: "dummy", BaseURL: server.URL})
			before := types.StringValue("sourceProject12345")
			after := types.StringValue("targetProject12345")
			prior := tfsdk.State{Schema: schema.Schema{Attributes: map[string]schema.Attribute{"project_id": schema.StringAttribute{Computed: true}}}}
			data := struct {
				ProjectID types.String `tfsdk:"project_id"`
			}{before}
			if d := prior.Set(context.Background(), &data); d.HasError() {
				t.Fatal(d)
			}
			state := prior
			var diags diag.Diagnostics
			if moveProject(context.Background(), c, false, "check12345678", before, after, prior, &state, &diags) {
				t.Fatal("rejected move reported success")
			}
			if !diags.HasError() || requests != 1 || !state.Raw.Equal(prior.Raw) {
				t.Fatal("rejection must preserve state")
			}
			if strings.Contains(fmt.Sprint(diags), "sensitive response") {
				t.Fatal("API response leaked")
			}
		})
	}
}

func TestProjectTokenScope(t *testing.T) {
	var response resource.SchemaResponse
	NewTokenResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	grants := response.Schema.Attributes["grants"].(schema.SetNestedAttribute)
	scope := grants.NestedObject.Attributes["scope"].(schema.StringAttribute)
	for _, v := range scope.Validators {
		var result validator.StringResponse
		v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue("PROJECTS")}, &result)
		if result.Diagnostics.HasError() {
			t.Fatal(result.Diagnostics)
		}
	}
}

func TestProjectMoveSavesCommittedOwnership(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"result":{"id":"check12345678","project_id":"targetProject12345"}}`)
	}))
	defer server.Close()
	c := client.NewClient(&client.Config{APIKey: "dummy", BaseURL: server.URL})
	before := types.StringValue("sourceProject12345")
	after := types.StringValue("targetProject12345")
	prior := tfsdk.State{Schema: schema.Schema{Attributes: map[string]schema.Attribute{"project_id": schema.StringAttribute{Computed: true}, "id": schema.StringAttribute{Computed: true}, "paused": schema.BoolAttribute{Computed: true}}}}
	data := struct {
		ProjectID types.String `tfsdk:"project_id"`
		ID        types.String `tfsdk:"id"`
		Paused    types.Bool   `tfsdk:"paused"`
	}{before, types.StringValue("check12345678"), types.BoolValue(true)}
	if d := prior.Set(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}
	var state tfsdk.State
	var diags diag.Diagnostics
	if !moveProject(context.Background(), c, false, "check12345678", before, after, prior, &state, &diags) {
		t.Fatal(diags)
	}
	if d := state.Get(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}
	if !data.ProjectID.Equal(after) || data.ID.ValueString() != "check12345678" || !data.Paused.ValueBool() {
		t.Fatal("committed ownership must retain other prior state for recovery")
	}
}
