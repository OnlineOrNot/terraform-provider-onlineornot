package client

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"testing"
)

func TestListWebhooksPagination(t *testing.T) {
	for _, tc := range []struct {
		name  string
		pages [][]Webhook
		total int
		want  []string
	}{
		{"empty", [][]Webhook{{}}, 0, []string{}},
		{"single page", [][]Webhook{{{ID: "one"}}}, 1, []string{"one"}},
		{"server caps page size", [][]Webhook{{{ID: "one"}}, {{ID: "two"}}, {{ID: "three"}}}, 3, []string{"one", "two", "three"}},
		{"empty page stops stale total", [][]Webhook{{{ID: "one"}}, {}}, 10, []string{"one"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server, apiClient := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodGet || r.URL.Path != "/v1/webhooks" || r.URL.Query().Get("page") != strconv.Itoa(requests) || r.URL.Query().Get("per_page") != "100" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if requests > len(tc.pages) {
					t.Error("requested a page after pagination should have stopped")
					http.Error(w, "unexpected page", 500)
					return
				}
				writeJSON(t, w, APIListResponse[Webhook]{Success: true, Result: tc.pages[requests-1], ResultInfo: ResultInfo{TotalCount: tc.total}})
			})
			defer server.Close()
			webhooks, err := apiClient.ListWebhooks()
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, 0, len(webhooks))
			for _, webhook := range webhooks {
				ids = append(ids, webhook.ID)
			}
			if !reflect.DeepEqual(ids, tc.want) {
				t.Errorf("got %v, want %v", ids, tc.want)
			}
			if requests != len(tc.pages) {
				t.Errorf("got %d requests, want %d", requests, len(tc.pages))
			}
		})
	}
}

func TestListWebhooksRejectsPartialResultsOnLaterPageFailure(t *testing.T) {
	for _, failure := range []string{"HTTP error", "API error", "invalid JSON"} {
		t.Run(failure, func(t *testing.T) {
			server, apiClient := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") == "1" {
					writeJSON(t, w, APIListResponse[Webhook]{Success: true, Result: []Webhook{{ID: "one"}}, ResultInfo: ResultInfo{TotalCount: 2}})
					return
				}
				switch failure {
				case "HTTP error":
					http.Error(w, "unavailable", 503)
				case "API error":
					writeJSON(t, w, APIListResponse[Webhook]{Errors: []APIError{{Message: "unavailable"}}})
				case "invalid JSON":
					fmt.Fprint(w, "not json")
				}
			})
			defer server.Close()
			webhooks, err := apiClient.ListWebhooks()
			if err == nil || webhooks != nil {
				t.Fatalf("expected error without partial results, got %v, %v", webhooks, err)
			}
		})
	}
}
