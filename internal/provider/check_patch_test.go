package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

func TestConfiguredCheckPatchSkipsUnknownAndOmitted(t *testing.T) {
	ctx := context.Background()
	r := &CheckResource{}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	var model checkModel
	var diagnostics diag.Diagnostics
	r.populateModelFromAPI(ctx, &model, &client.Check{Name: "API"}, &diagnostics)
	model.FollowRedirects = types.BoolUnknown()
	model.ConfirmationPeriodSeconds = types.Int64Unknown()
	model.RecoveryPeriodSeconds = types.Int64Value(0)
	model.ReminderAlertIntervalMinutes = types.Int64Null()
	plan := tfsdk.Plan{Schema: schema.Schema}
	diagnostics.Append(plan.Set(ctx, &model)...)
	check := checkModelToClient(ctx, &model, "", &diagnostics)
	fields := configuredCheckPatch(tfsdk.Config{Raw: plan.Raw, Schema: plan.Schema}, check, &diagnostics)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	for _, key := range []string{"follow_redirects", "confirmation_period_seconds", "reminder_alert_interval_minutes"} {
		if _, ok := fields[key]; ok {
			t.Errorf("sent unknown/omitted %s", key)
		}
	}
	if fields["recovery_period_seconds"] != 0 {
		t.Errorf("lost explicit zero: %v", fields)
	}
}

func TestComponentUpdateUsesPriorIdentityWithUnknownPlanID(t *testing.T) {
	ctx := context.Background()
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "PATCH" || r.URL.Path != "/v1/status_pages/page1234/components/comp1234" {
			t.Errorf("bad update route: %s %s", r.Method, r.URL.Path)
		}
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if _, ok := input["status"]; ok {
			t.Error("unknown status must not be sent")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": "comp1234", "status_page_id": "page1234", "name": "Renamed", "status": "MAJOR_OUTAGE", "display_uptime": true, "display_metrics": true}})
	}))
	defer server.Close()
	r := &StatusPageComponentResource{client: client.NewClient(&client.Config{APIKey: "loopback-only", BaseURL: server.URL})}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	model := statusPageComponentModel{Id: types.StringValue("comp1234"), StatusPageId: types.StringValue("page1234"), Name: types.StringValue("API"), Status: types.StringValue("MAJOR_OUTAGE"), DisplayUptime: types.BoolValue(true), DisplayMetrics: types.BoolValue(true), CheckIds: types.ListNull(types.StringType)}
	state := tfsdk.State{Schema: schema.Schema}
	if d := state.Set(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	model.Id = types.StringUnknown()
	model.StatusPageId = types.StringUnknown()
	model.Name = types.StringValue("Renamed")
	model.Status = types.StringUnknown()
	plan := tfsdk.Plan{Schema: schema.Schema}
	if d := plan.Set(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	response := frameworkresource.UpdateResponse{State: tfsdk.State{Schema: schema.Schema}}
	r.Update(ctx, frameworkresource.UpdateRequest{Plan: plan, State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var result statusPageComponentModel
	if d := response.State.Get(ctx, &result); d.HasError() {
		t.Fatal(d)
	}
	if result.Id.ValueString() != "comp1234" || result.StatusPageId.ValueString() != "page1234" {
		t.Errorf("identity not restored: %v", result)
	}
	if !called {
		t.Fatal("no update request")
	}
}
