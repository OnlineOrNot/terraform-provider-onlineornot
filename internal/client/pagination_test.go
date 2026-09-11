package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestListPagination(t *testing.T) {
	t.Run("checks", func(t *testing.T) {
		testListPagination(t, "/v1/checks", (*Client).ListChecks)
	})
	t.Run("heartbeats", func(t *testing.T) {
		testListPagination(t, "/v1/heartbeats", (*Client).ListHeartbeats)
	})
	t.Run("maintenance windows", func(t *testing.T) {
		testListPagination(t, "/v1/maintenance-windows", (*Client).ListMaintenanceWindows)
	})
	t.Run("status pages", func(t *testing.T) {
		testListPagination(t, "/v1/status_pages", (*Client).ListStatusPages)
	})
	t.Run("users", func(t *testing.T) {
		testListPagination(t, "/v1/users", (*Client).ListUsers)
	})
	t.Run("webhooks", func(t *testing.T) {
		testListPagination(t, "/v1/webhooks", (*Client).ListWebhooks)
	})
	t.Run("components", func(t *testing.T) {
		testListPagination(t, "/v1/status_pages/page-123/components", func(c *Client) ([]StatusPageComponent, error) {
			return c.ListStatusPageComponents("page-123")
		})
	})
	t.Run("groups", func(t *testing.T) {
		testListPagination(t, "/v1/status_pages/page-123/groups", func(c *Client) ([]StatusPageComponentGroup, error) {
			return c.ListStatusPageComponentGroups("page-123")
		})
	})
	t.Run("incidents", func(t *testing.T) {
		testListPagination(t, "/v1/status_pages/page-123/incidents", func(c *Client) ([]StatusPageIncident, error) {
			return c.ListStatusPageIncidents("page-123")
		})
	})
}

// Exercise the wire contract through every public list method, not just the helper.
func testListPagination[T any](t *testing.T, path string, list func(*Client) ([]T, error)) {
	t.Helper()
	for _, tc := range []struct {
		name     string
		sizes    []int
		total    int
		perPage  int
		metadata string
	}{
		{"empty dataset", []int{0}, 0, 100, "numeric"},
		{"single page", []int{1}, 1, 100, "numeric"},
		{"multiple pages", []int{100, 100, 1}, 201, 100, "numeric"},
		{"exact final page", []int{100, 100}, 200, 100, "numeric"},
		{"server default page size", []int{20, 20, 1}, 41, 20, "numeric"},
		{"server caps page size", []int{1, 1, 1}, 3, 1, "numeric"},
		{"quoted page and per_page", []int{1, 1, 1}, 3, 1, "quoted"},
		{"total only metadata", []int{1, 1}, 2, 100, "total only"},
		{"empty page stops stale total", []int{1, 0}, 10, 100, "numeric"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			want := make([]T, 0)
			server, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assertListRequest(t, r, path, requests)
				if requests > len(tc.sizes) {
					t.Error("requested a page after pagination should have stopped")
					http.Error(w, "unexpected page", http.StatusInternalServerError)
					return
				}
				items := make([]map[string]string, tc.sizes[requests-1])
				for i := range items {
					items[i] = map[string]string{"id": fmt.Sprintf("item-%d", len(want)+i)}
				}
				encoded, err := json.Marshal(items)
				if err != nil {
					t.Error(err)
					return
				}
				var typedItems []T
				if err := json.Unmarshal(encoded, &typedItems); err != nil {
					t.Error(err)
					return
				}
				want = append(want, typedItems...)
				info := map[string]any{"total_count": tc.total}
				if tc.metadata != "total only" {
					info["page"], info["per_page"], info["count"] = requests, tc.perPage, len(items)
				}
				if tc.metadata == "quoted" {
					info["page"], info["per_page"] = strconv.Itoa(requests), strconv.Itoa(tc.perPage)
				}
				writeJSON(t, w, map[string]any{"success": true, "result": items, "result_info": info})
			})
			defer server.Close()
			got, err := list(c)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %#v, want %#v", got, want)
			}
			if requests != len(tc.sizes) {
				t.Errorf("got %d requests, want %d", requests, len(tc.sizes))
			}
		})
	}

	type failureCase struct {
		name   string
		status int
		body   string
		want   string
	}
	failures := []failureCase{
		{"HTTP error", 503, "unavailable", "API request failed with status 503"},
		{"API error", 200, `{"success":false,"errors":[{"message":"unavailable"}]}`, "API error: unavailable"},
		{"API failure without errors", 200, `{"success":false}`, "API request failed"},
		{"invalid JSON", 200, "not json", "failed to parse response"},
		{"invalid result", 200, `{"success":true,"result":{},"result_info":{"total_count":2}}`, "failed to parse response"},
		{"missing metadata", 200, `{"success":true,"result":[]}`, "invalid pagination total_count"},
		{"null metadata", 200, `{"success":true,"result":[],"result_info":null}`, "invalid pagination total_count"},
		{"nonempty result with zero total", 200, `{"success":true,"result":[{"id":"two"}],"result_info":{"total_count":0}}`, "pagination results exceed total_count"},
		{"results exceed total", 200, `{"success":true,"result":[{"id":"two"},{"id":"three"}],"result_info":{"total_count":1}}`, "pagination results exceed total_count"},
	}
	for _, metadata := range []struct {
		name string
		json string
		want string
	}{
		{"missing total", `{}`, "invalid pagination total_count"},
		{"null total", `{"total_count":null}`, "invalid pagination total_count"},
		{"negative total", `{"total_count":-1}`, "invalid pagination total_count"},
		{"fractional total", `{"total_count":1.5}`, "failed to parse response"},
		{"invalid total type", `{"total_count":{}}`, "failed to parse response"},
		{"wrong page", `{"total_count":2,"page":99}`, "invalid pagination page"},
		{"fractional page", `{"total_count":2,"page":"1.5"}`, "invalid pagination page"},
		{"invalid page", `{"total_count":2,"page":"oops"}`, "failed to parse response"},
		{"overflowing page", `{"total_count":2,"page":"9223372036854775808"}`, "invalid pagination page"},
		{"zero page size", `{"total_count":2,"per_page":0}`, "invalid pagination per_page"},
		{"negative page size", `{"total_count":2,"per_page":-1}`, "invalid pagination per_page"},
		{"fractional page size", `{"total_count":2,"per_page":"1.5"}`, "invalid pagination per_page"},
		{"invalid page size", `{"total_count":2,"per_page":true}`, "failed to parse response"},
		{"wrong count", `{"total_count":2,"count":10}`, "invalid pagination count"},
		{"invalid count type", `{"total_count":2,"count":{}}`, "failed to parse response"},
	} {
		failures = append(failures, failureCase{metadata.name, 200, `{"success":true,"result":[],"result_info":` + metadata.json + `}`, metadata.want})
	}
	for _, failure := range failures {
		for _, failPage := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s on page %d", failure.name, failPage), func(t *testing.T) {
				requests := 0
				server, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
					requests++
					assertListRequest(t, r, path, requests)
					if requests < failPage {
						fmt.Fprint(w, `{"success":true,"result":[{"id":"one"}],"result_info":{"page":"1","per_page":"100","count":1,"total_count":2}}`)
						return
					}
					w.WriteHeader(failure.status)
					fmt.Fprint(w, failure.body)
				})
				defer server.Close()
				got, err := list(c)
				if err == nil || got != nil {
					t.Fatalf("expected error without partial results, got %v, %v", got, err)
				}
				if !strings.Contains(err.Error(), failure.want) {
					t.Errorf("got error %q, want %q", err, failure.want)
				}
				if failure.status == 503 {
					var httpErr *HTTPError
					if !errors.As(err, &httpErr) || httpErr.StatusCode != 503 {
						t.Errorf("HTTP error type/status not preserved: %v", err)
					}
				}
				if requests != failPage {
					t.Errorf("got %d requests, want %d", requests, failPage)
				}
			})
		}
	}
}

func assertListRequest(t *testing.T, r *http.Request, path string, page int) {
	t.Helper()
	if r.Method != http.MethodGet || r.URL.Path != path || r.URL.Query().Get("page") != strconv.Itoa(page) || r.URL.Query().Get("per_page") != "100" {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
	}
}

func TestListAllPreservesQuery(t *testing.T) {
	requests := 0
	server, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assertListRequest(t, r, "/v1/status_pages/page-123/components", requests)
		query := r.URL.Query()
		if query.Get("search") != "a & b" || !reflect.DeepEqual(query["filter"], []string{"one", "two"}) || len(query["page"]) != 1 || len(query["per_page"]) != 1 {
			t.Errorf("query not preserved: %v", query)
		}
		if requests > 2 {
			http.Error(w, "unexpected page", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, `{"success":true,"result":[{"id":"item-%d"}],"result_info":{"total_count":2}}`, requests)
	})
	defer server.Close()
	items, err := listAll[StatusPageComponent](c, "/v1/status_pages/page-123/components?search=a+%26+b&filter=one&filter=two&page=9&per_page=1")
	if err != nil || len(items) != 2 || requests != 2 {
		t.Fatalf("got %v, %v, %d requests", items, err, requests)
	}
}
