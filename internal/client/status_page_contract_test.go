package client

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("failed to encode response: %v", err)
	}
}

func TestStatusPageComponentGroupPathsMatchOpenAPI(t *testing.T) {
	expected := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/status_pages/page1234/groups"},
		{http.MethodGet, "/v1/status_pages/page1234/groups/group123"},
		{http.MethodPatch, "/v1/status_pages/page1234/groups/group123"},
		{http.MethodDelete, "/v1/status_pages/page1234/groups/group123"},
		{http.MethodGet, "/v1/status_pages/page1234/groups"},
	}
	request := 0
	server, apiClient := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if request >= len(expected) {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := expected[request]
		request++
		if r.Method != want.method || r.URL.Path != want.path {
			t.Errorf("expected %s %s, got %s %s", want.method, want.path, r.Method, r.URL.Path)
		}
		if request == len(expected) {
			writeJSON(t, w, APIListResponse[StatusPageComponentGroup]{Success: true})
			return
		}
		writeJSON(t, w, APIResponse[StatusPageComponentGroup]{
			Result:  StatusPageComponentGroup{ID: "group123", Name: "Core"},
			Success: true,
		})
	})
	defer server.Close()

	group := &StatusPageComponentGroup{Name: "Core"}
	if _, err := apiClient.CreateStatusPageComponentGroup("page1234", group); err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.GetStatusPageComponentGroup("page1234", "group123"); err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.UpdateStatusPageComponentGroup("page1234", "group123", group); err != nil {
		t.Fatal(err)
	}
	if err := apiClient.DeleteStatusPageComponentGroup("page1234", "group123"); err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.ListStatusPageComponentGroups("page1234"); err != nil {
		t.Fatal(err)
	}
	if request != len(expected) {
		t.Fatalf("expected %d requests, got %d", len(expected), request)
	}
}

func TestUpdateStatusPageComponentMatchesOpenAPIPayload(t *testing.T) {
	displayUptime := true
	displayMetrics := false
	var groupID *string
	checkIDs := []string{}
	var heartbeatID *string
	overrideStatus := false
	server, apiClient := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/status_pages/page1234/components/comp1234" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		expected := map[string]any{
			"name":            "API",
			"status":          "OPERATIONAL",
			"display_uptime":  true,
			"display_metrics": false,
			"group_id":        nil,
			"check_ids":       []any{},
			"heartbeat_id":    nil,
			"override_status": false,
		}
		if !reflect.DeepEqual(body, expected) {
			t.Errorf("unexpected component patch\nwant: %#v\n got: %#v", expected, body)
		}
		writeJSON(t, w, APIResponse[StatusPageComponent]{
			Result:  StatusPageComponent{ID: "comp1234", Name: "API", Status: "OPERATIONAL"},
			Success: true,
		})
	})
	defer server.Close()

	_, err := apiClient.UpdateStatusPageComponent("page1234", "comp1234", &StatusPageComponentPatch{
		Name:           "API",
		Status:         "OPERATIONAL",
		DisplayUptime:  &displayUptime,
		DisplayMetrics: &displayMetrics,
		GroupID:        &groupID,
		CheckIDs:       &checkIDs,
		HeartbeatID:    &heartbeatID,
		OverrideStatus: &overrideStatus,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateStatusPageComponentOmitsUnmanagedRelationships(t *testing.T) {
	server, apiClient := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		expected := map[string]any{"name": "API"}
		if !reflect.DeepEqual(body, expected) {
			t.Errorf("unexpected component patch\nwant: %#v\n got: %#v", expected, body)
		}
		writeJSON(t, w, APIResponse[StatusPageComponent]{
			Result:  StatusPageComponent{ID: "comp1234", Name: "API", Status: "OPERATIONAL"},
			Success: true,
		})
	})
	defer server.Close()

	if _, err := apiClient.UpdateStatusPageComponent("page1234", "comp1234", &StatusPageComponentPatch{Name: "API"}); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateStatusPageIncidentMatchesOpenAPIPayload(t *testing.T) {
	impact := "MAJOR_OUTAGE"
	server, apiClient := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/status_pages/page1234/incidents/inc12345" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		expected := map[string]any{"title": "API outage", "impact": "MAJOR_OUTAGE"}
		if !reflect.DeepEqual(body, expected) {
			t.Errorf("unexpected incident patch\nwant: %#v\n got: %#v", expected, body)
		}
		writeJSON(t, w, APIResponse[StatusPageIncident]{
			Result:  StatusPageIncident{ID: "inc12345", Title: "API outage", Impact: &impact},
			Success: true,
		})
	})
	defer server.Close()

	_, err := apiClient.UpdateStatusPageIncident("page1234", "inc12345", &StatusPageIncidentPatch{
		Title:  "API outage",
		Impact: &impact,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateStatusPageScheduledMaintenanceMatchesOpenAPIPayload(t *testing.T) {
	server, apiClient := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/status_pages/page1234/scheduled_maintenance/main1234" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		expected := map[string]any{
			"title":            "Database upgrade",
			"start_date":       "2026-09-05T02:00:00Z",
			"duration_minutes": float64(120),
		}
		if !reflect.DeepEqual(body, expected) {
			t.Errorf("unexpected scheduled maintenance patch\nwant: %#v\n got: %#v", expected, body)
		}
		writeJSON(t, w, APIResponse[StatusPageScheduledMaintenance]{Success: true})
	})
	defer server.Close()

	_, err := apiClient.UpdateStatusPageScheduledMaintenance("page1234", "main1234", &StatusPageScheduledMaintenancePatch{
		Title:           "Database upgrade",
		StartDate:       "2026-09-05T02:00:00Z",
		DurationMinutes: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
}
