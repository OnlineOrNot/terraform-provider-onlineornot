package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

func TestHeartbeatReadHTTPErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{401, 403, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" || req.URL.Path != "/v1/heartbeats/heartbeat1" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				fmt.Fprint(w, `{"success":false,"result":null,"errors":[{"code":10006,"message":"Heartbeat not found or unavailable"}],"messages":[]}`)
			}))
			defer server.Close()
			r := &HeartbeatResource{client: client.NewClient(&client.Config{BaseURL: server.URL, APIKey: "fixture"})}
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			state := tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)}
			if d := state.SetAttribute(ctx, path.Root("id"), types.StringValue("heartbeat1")); d.HasError() {
				t.Fatal(d)
			}
			response := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &response)
			if status == 404 {
				if response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
					t.Fatalf("404 should remove resource: %v", response.Diagnostics)
				}
			} else if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
				t.Fatalf("HTTP %d must report an error and preserve state: %v", status, response.Diagnostics)
			}
		})
	}
}

// Exercise import and successive refreshes through the real HTTP client, rather
// than only testing model conversion. Every request must remain read-only.
func TestHeartbeatImportAndRefresh(t *testing.T) {
	ctx := context.Background()
	alertFields := []string{"user_alerts", "slack_alerts", "webhook_alerts", "discord_alerts", "oncall_alerts", "incident_io_alerts", "microsoft_teams_alerts", "telegram_alerts", "pushover_alerts"}
	for _, schedule := range []string{"interval", "cron"} {
		t.Run(schedule, func(t *testing.T) {
			remote := map[string]any{"id": "heartbeat1", "name": "Backup", "grace_period": 60, "alert_priority": "LOW", "reminder_alert_interval_minutes": 1440, "status": "ACTIVE"}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/v1/heartbeats/heartbeat1" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]any{"success": true, "result": remote}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			r := &HeartbeatResource{client: client.NewClient(&client.Config{BaseURL: server.URL, APIKey: "fixture"})}
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			imported := resource.ImportStateResponse{State: tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "heartbeat1"}, &imported)
			if imported.Diagnostics.HasError() {
				t.Fatal(imported.Diagnostics)
			}
			state := imported.State
			for _, stage := range []string{"import", "unchanged", "remote change", "schedule switch", "empty", "null", "omitted"} {
				t.Run(stage, func(t *testing.T) {
					expected := map[string]attr.Value{}
					if stage == "import" || stage == "remote change" {
						period, cron, timezone := 300, "*/5 * * * *", "Europe/London"
						if stage == "remote change" {
							period, cron, timezone = 600, "*/10 * * * *", "Australia/Sydney"
						}
						if schedule == "interval" {
							remote["report_period"] = period
							remote["report_period_cron"] = nil
							remote["timezone"] = nil
						} else {
							remote["report_period"] = nil
							remote["report_period_cron"] = cron
							remote["timezone"] = timezone
						}
						for _, field := range alertFields {
							remote[field] = []string{field + "-" + stage}
						}
					}
					if stage == "schedule switch" {
						if schedule == "interval" {
							remote["report_period"], remote["report_period_cron"], remote["timezone"] = nil, "0 * * * *", "UTC"
						} else {
							remote["report_period"], remote["report_period_cron"], remote["timezone"] = 3600, nil, nil
						}
					}
					if stage == "empty" || stage == "null" || stage == "omitted" {
						remote["report_period"], remote["report_period_cron"], remote["timezone"] = nil, "", ""
						for _, field := range alertFields {
							switch stage {
							case "empty":
								remote[field] = []string{}
							case "null":
								remote[field] = nil
							case "omitted":
								delete(remote, field)
							}
						}
					}
					expected["report_period"] = types.Int64Null()
					if v, ok := remote["report_period"].(int); ok {
						expected["report_period"] = types.Int64Value(int64(v))
					}
					for _, field := range []string{"report_period_cron", "timezone"} {
						expected[field] = types.StringNull()
						if v, ok := remote[field].(string); ok && v != "" {
							expected[field] = types.StringValue(v)
						}
					}
					for _, field := range alertFields {
						expected[field] = types.ListNull(types.StringType)
						if v, ok := remote[field].([]string); ok {
							list, d := types.ListValueFrom(ctx, types.StringType, v)
							if d.HasError() {
								t.Fatal(d)
							}
							expected[field] = list
						}
					}
					response := resource.ReadResponse{State: state}
					r.Read(ctx, resource.ReadRequest{State: state}, &response)
					if response.Diagnostics.HasError() {
						t.Fatal(response.Diagnostics)
					}
					for field, want := range expected {
						var got attr.Value
						if d := response.State.GetAttribute(ctx, path.Root(field), &got); d.HasError() {
							t.Fatal(d)
						}
						if !got.Equal(want) {
							t.Errorf("%s = %s, want %s", field, got, want)
						}
					}
					if stage == "unchanged" && !response.State.Raw.Equal(state.Raw) {
						t.Error("unchanged refresh modified state")
					}
					state = response.State
				})
			}
		})
	}
}
