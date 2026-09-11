package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_webhook"
)

func webhookTestList(t *testing.T, ids ...string) types.List {
	t.Helper()
	if ids == nil {
		ids = []string{}
	}
	value, ds := types.ListValueFrom(context.Background(), types.StringType, ids)
	if ds.HasError() {
		t.Fatal(ds)
	}
	return value
}

func webhookState(t *testing.T, model resource_webhook.WebhookModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: resource_webhook.WebhookResourceSchema(context.Background())}
	if ds := state.Set(context.Background(), &model); ds.HasError() {
		t.Fatal(ds)
	}
	return state
}

func TestWebhookResourceRoundtrip(t *testing.T) {
	ctx := context.Background()
	list := func(ids ...string) types.List { return webhookTestList(t, ids...) }
	model := resource_webhook.WebhookModel{Id: types.StringUnknown(), Url: types.StringValue("https://example.com/hook"), Description: types.StringNull(), Events: list("uptime.up", "uptime.down"), CheckIds: list("check002", "check001"), HeartbeatIds: list("heartbeat1"), StatusPageIds: list("status01")}
	associations := map[string][]string{"checks": {"check001", "check002"}, "heartbeats": {"heartbeat1"}, "status_pages": {"status01"}}
	events := []string{"uptime.down", "uptime.up"}
	description := any(nil)
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == "POST" || r.Method == "PATCH" {
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			for response, request := range map[string]string{"checks": "check_ids", "heartbeats": "heartbeat_ids", "status_pages": "status_page_ids"} {
				raw, ok := body[request]
				if !ok {
					t.Errorf("missing %s", request)
					continue
				}
				var ids []string
				if err := json.Unmarshal(raw, &ids); err != nil {
					t.Error(err)
				}
				associations[response] = ids
			}
			if raw, ok := body["description"]; ok {
				if err := json.Unmarshal(raw, &description); err != nil {
					t.Error(err)
				}
			}
		}
		refs := func(ids []string) []map[string]string {
			result := make([]map[string]string, len(ids))
			for i, id := range ids {
				result[len(ids)-i-1] = map[string]string{"id": id, "name": "Example"}
			}
			return result
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": "webhook1", "url": "https://example.com/hook", "description": description, "events": events, "checks": refs(associations["checks"]), "heartbeats": refs(associations["heartbeats"]), "status_pages": refs(associations["status_pages"])}})
	}))
	defer server.Close()
	r := &WebhookResource{client: client.NewClient(&client.Config{BaseURL: server.URL})}
	state := webhookState(t, model)
	created := resource.CreateResponse{State: state}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	model.Id = types.StringValue("webhook1")
	expected := webhookState(t, model)
	if !created.State.Raw.Equal(expected.Raw) {
		t.Fatalf("create did not preserve plan: %s", created.State.Raw)
	}
	read := resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if !read.State.Raw.Equal(created.State.Raw) {
		t.Fatalf("no-op refresh changed state: %s", read.State.Raw)
	}
	// Remote membership/event drift must be refreshed, not retained from old state.
	associations["checks"] = []string{"check003"}
	associations["heartbeats"] = []string{}
	associations["status_pages"] = []string{"status02"}
	events = []string{"heartbeat.down"}
	r.Read(ctx, resource.ReadRequest{State: read.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	var drift resource_webhook.WebhookModel
	if ds := read.State.Get(ctx, &drift); ds.HasError() {
		t.Fatal(ds)
	}
	if !drift.CheckIds.Equal(list("check003")) || !drift.HeartbeatIds.Equal(list()) || !drift.Events.Equal(list("heartbeat.down")) || !drift.StatusPageIds.Equal(list("status02")) {
		t.Fatalf("drift not reflected: %#v", drift)
	}
	// Every association can be explicitly cleared. Empty description must be sent too.
	model.CheckIds, model.HeartbeatIds, model.StatusPageIds = list(), list(), list()
	model.Events = list("heartbeat.down")
	model.Description = types.StringValue("")
	state = webhookState(t, model)
	updated := resource.UpdateResponse{State: read.State}
	r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}, State: read.State}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}
	if !updated.State.Raw.Equal(state.Raw) {
		t.Fatalf("clear state = %s", updated.State.Raw)
	}
	r.Read(ctx, resource.ReadRequest{State: updated.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if !read.State.Raw.Equal(updated.State.Raw) {
		t.Fatal("empty refresh changed state")
	}
	if !reflect.DeepEqual(methods, []string{"POST", "GET", "GET", "PATCH", "GET"}) {
		t.Fatalf("requests = %v", methods)
	}
}

func TestWebhookImportAndMissing(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{200, 404, 403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"success":true,"result":{"id":"webhook1","url":"https://example.com/hook","description":null,"events":["uptime.down"],"checks":[{"id":"check001","name":"API"}],"heartbeats":[],"status_pages":[{"id":"status01","name":"Status"}]}}`)
			}))
			defer server.Close()
			r := &WebhookResource{client: client.NewClient(&client.Config{BaseURL: server.URL})}
			state := webhookState(t, resource_webhook.WebhookModel{Id: types.StringValue("webhook1"), CheckIds: types.ListNull(types.StringType), HeartbeatIds: types.ListNull(types.StringType), StatusPageIds: types.ListNull(types.StringType), Events: types.ListNull(types.StringType)})
			response := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &response)
			if status == 403 || status == 500 {
				if !response.Diagnostics.HasError() || response.State.Raw.IsNull() {
					t.Fatal("error must retain state and diagnose")
				}
				return
			}
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			if status == 404 {
				if !response.State.Raw.IsNull() {
					t.Fatal("missing resource retained")
				}
				deleted := resource.DeleteResponse{}
				r.Delete(ctx, resource.DeleteRequest{State: state}, &deleted)
				if deleted.Diagnostics.HasError() {
					t.Fatal(deleted.Diagnostics)
				}
				return
			}
			var got resource_webhook.WebhookModel
			if ds := response.State.Get(ctx, &got); ds.HasError() {
				t.Fatal(ds)
			}
			if !got.CheckIds.Equal(webhookTestList(t, "check001")) || !got.StatusPageIds.Equal(webhookTestList(t, "status01")) || !got.Events.Equal(webhookTestList(t, "uptime.down")) || !got.HeartbeatIds.IsNull() {
				t.Fatalf("import state = %#v", got)
			}
		})
	}
}

func TestWebhookComputedAssociations(t *testing.T) {
	ctx := context.Background()
	for _, value := range []types.List{types.ListNull(types.StringType), types.ListUnknown(types.StringType)} {
		model := resource_webhook.WebhookModel{Url: types.StringValue("https://example.com/hook"), Events: webhookTestList(t, "uptime.down"), Description: types.StringUnknown(), CheckIds: value, HeartbeatIds: value, StatusPageIds: value}
		request, ds := webhookRequestFromModel(ctx, &model)
		if ds.HasError() {
			t.Fatal(ds)
		}
		if request.Description != nil || request.CheckIDs != nil || request.HeartbeatIDs != nil || request.StatusPageIDs != nil {
			t.Fatalf("unset fields sent: %#v", request)
		}
		response := &client.Webhook{ID: "webhook1", URL: request.URL, Events: request.Events, CheckIDs: []string{"check001"}, HeartbeatIDs: []string{}, StatusPageIDs: []string{}}
		if ds := populateWebhookModel(ctx, &model, response); ds.HasError() {
			t.Fatal(ds)
		}
		if !model.CheckIds.Equal(webhookTestList(t, "check001")) || model.HeartbeatIds.IsUnknown() || model.StatusPageIds.IsUnknown() || model.Description.IsUnknown() {
			t.Fatalf("computed state = %#v", model)
		}
		before := webhookState(t, model)
		if ds := populateWebhookModel(ctx, &model, response); ds.HasError() {
			t.Fatal(ds)
		}
		after := webhookState(t, model)
		if !before.Raw.Equal(after.Raw) {
			t.Fatal("computed refresh changed state")
		}
	}
}
