package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProjectMoveAcknowledgements(t *testing.T) {
	const id = "monitor123"
	const target = "targetProject1234567"
	for _, kind := range []string{"checks", "heartbeats"} {
		for _, tc := range []struct {
			name   string
			result map[string]any
			valid  bool
		}{
			{"matching", map[string]any{"id": id, "project_id": target}, true},
			{"NULL assignment", map[string]any{"id": id, "project_id": nil}, false},
			{"omitted assignment", map[string]any{"id": id}, false},
			{"wrong assignment", map[string]any{"id": id, "project_id": "sourceProject1234567"}, false},
			{"wrong resource", map[string]any{"id": "different123", "project_id": target}, false},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" || r.URL.Path != "/v1/"+kind+"/"+id+"/move" || r.Header.Get("Authorization") != "Bearer loopback-only" {
						t.Errorf("unexpected move request %s %s", r.Method, r.URL)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if len(body) != 1 || body["project_id"] != target {
						t.Errorf("unexpected move body %v", body)
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]any{"success": true, "result": tc.result})
				}))
				defer server.Close()
				api := NewClient(&Config{BaseURL: server.URL, APIKey: "loopback-only"})
				move := api.MoveCheck
				if kind == "heartbeats" {
					move = api.MoveHeartbeat
				}
				err := move(id, target)
				if tc.valid {
					if err != nil {
						t.Fatal(err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "invalid project move acknowledgement") {
					t.Fatalf("expected rejected move acknowledgement, got %v", err)
				}
			})
		}
	}
}
