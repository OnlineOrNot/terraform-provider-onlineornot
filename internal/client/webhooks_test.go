package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

// Deliberately not encoded from Webhook: these are the public API's response keys.
const webhookAPIResult = `{"id":"webhook1","url":"https://example.com/hook","description":null,"events":["uptime.down"],"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","checks":[{"id":"check001","name":"API"}],"heartbeats":[{"id":"heartbeat1","name":"Backup"}],"status_pages":[{"id":"status01","name":"Status"}]}`

func TestWebhookWireContracts(t *testing.T) {
	for _, method := range []string{"POST", "GET", "PATCH", "LIST", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			server, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				expectedMethod, expectedPath := method, "/v1/webhooks/webhook1"
				if method == "POST" || method == "LIST" {
					expectedPath = "/v1/webhooks"
				}
				if method == "LIST" {
					expectedMethod = "GET"
				}
				if r.Method != expectedMethod || r.URL.Path != expectedPath {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				if method == "POST" || method == "PATCH" {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					want := map[string]any{"url": "https://example.com/hook", "description": "", "events": []any{"uptime.down"}, "check_ids": []any{"check001"}, "heartbeat_ids": []any{}, "status_page_ids": []any{}}
					if !reflect.DeepEqual(body, want) {
						t.Errorf("body = %#v, want %#v", body, want)
					}
				}
				result := webhookAPIResult
				if method == "LIST" {
					result = "[" + result + "]"
				}
				if method == "DELETE" {
					result = `{"id":"webhook1"}`
				}
				fmt.Fprintf(w, `{"success":true,"result":%s,"errors":[],"messages":[],"result_info":{"page":1,"per_page":100,"count":1,"total_count":1}}`, result)
			})
			defer server.Close()
			description := ""
			checks, empty := []string{"check001"}, []string{}
			request := &WebhookRequest{URL: "https://example.com/hook", Description: &description, Events: []string{"uptime.down"}, CheckIDs: &checks, HeartbeatIDs: &empty, StatusPageIDs: &empty}
			var got *Webhook
			var err error
			switch method {
			case "POST":
				got, err = c.CreateWebhook(request)
			case "PATCH":
				got, err = c.UpdateWebhook("webhook1", request)
			case "GET":
				got, err = c.GetWebhook("webhook1")
			case "LIST":
				var list []Webhook
				list, err = c.ListWebhooks()
				if len(list) != 1 {
					t.Fatalf("list = %#v, err = %v", list, err)
				}
				got = &list[0]
			case "DELETE":
				err = c.DeleteWebhook("webhook1")
			}
			if err != nil {
				t.Fatal(err)
			}
			if method != "DELETE" && (got.ID != "webhook1" || got.URL != request.URL || got.Description != "" || !reflect.DeepEqual(got.Events, request.Events) || !reflect.DeepEqual(got.CheckIDs, checks) || !reflect.DeepEqual(got.HeartbeatIDs, []string{"heartbeat1"}) || !reflect.DeepEqual(got.StatusPageIDs, []string{"status01"})) {
				t.Fatalf("decoded response = %#v", got)
			}
		})
	}
}

func TestWebhookOmittedAndEmptyAssociations(t *testing.T) {
	server, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body) != 0 {
			t.Errorf("omitted PATCH fields = %#v", body)
		}
		fmt.Fprint(w, `{"success":true,"result":{"id":"webhook1","url":"https://example.com/hook","description":null,"events":["uptime.down"],"checks":[],"heartbeats":[],"status_pages":[]}}`)
	})
	defer server.Close()
	got, err := c.UpdateWebhook("webhook1", &WebhookRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.CheckIDs == nil || len(got.CheckIDs) != 0 || got.HeartbeatIDs == nil || len(got.HeartbeatIDs) != 0 || got.StatusPageIDs == nil || len(got.StatusPageIDs) != 0 {
		t.Fatalf("empty associations = %#v", got)
	}
}

func TestWebhookHTTPErrors(t *testing.T) {
	for _, status := range []int{404, 403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"success":false,"errors":[{"code":1000,"message":"request failed"}],"messages":[]}`)
			})
			defer server.Close()
			operations := []func() error{
				func() error { _, err := c.CreateWebhook(&WebhookRequest{}); return err },
				func() error { _, err := c.GetWebhook("webhook1"); return err },
				func() error { _, err := c.UpdateWebhook("webhook1", &WebhookRequest{}); return err },
				func() error { _, err := c.ListWebhooks(); return err },
				func() error { return c.DeleteWebhook("webhook1") },
			}
			for _, operation := range operations {
				err := operation()
				if err == nil || IsNotFound(err) != (status == 404) {
					t.Fatalf("status %d: %v", status, err)
				}
			}
		})
	}
}
