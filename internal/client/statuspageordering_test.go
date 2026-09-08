package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
)

func TestOrderPaginationAndScope(t *testing.T) {
	for _, kind := range []string{"components", "groups", "group_components"} {
		t.Run(kind, func(t *testing.T) {
			pages := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pages++
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				if r.URL.Query().Get("per_page") != "100" || r.Method != "GET" {
					t.Errorf("unexpected request %s", r.URL)
				}
				wantPath := "/v1/status_pages/page1234/components"
				if kind == "groups" {
					wantPath = "/v1/status_pages/page1234/groups"
				}
				if r.URL.Path != wantPath {
					t.Errorf("path: %s", r.URL.Path)
				}
				// Simulate server clamping the requested page size to 20.
				start, end := (page-1)*20, page*20
				if end > 45 {
					end = 45
				}
				items := []map[string]any{}
				for i := start; i < end; i++ {
					group := any(nil)
					if i%2 == 1 {
						group = "group123"
					}
					items = append(items, map[string]any{"id": fmt.Sprintf("id%03d", 100-i), "group_id": group})
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "result": items, "result_info": map[string]int{"page": page, "count": len(items), "total_count": 45}})
			}))
			defer server.Close()
			c := NewClient(&Config{BaseURL: server.URL})
			scope := StatusPageOrderScope{PageID: "page1234", Kind: kind}
			if kind == "group_components" {
				scope.GroupID = "group123"
			}
			got, err := c.ListStatusPageOrder(scope)
			if err != nil {
				t.Fatal(err)
			}
			want := []string{}
			for i := 0; i < 45; i++ {
				if kind == "groups" || (kind == "components" && i%2 == 0) || (kind == "group_components" && i%2 == 1) {
					want = append(want, fmt.Sprintf("id%03d", 100-i))
				}
			}
			if !reflect.DeepEqual(got, want) || pages != 3 {
				t.Fatalf("got %v pages %d", got, pages)
			}
		})
	}
}

func TestOrderListRejectsIncompleteResponses(t *testing.T) {
	for name, body := range map[string]string{
		"missing metadata": `{"success":true,"result":[]}`,
		"missing total":    `{"success":true,"result":[],"result_info":{"count":0}}`,
		"no progress":      `{"success":true,"result":[],"result_info":{"count":0,"total_count":1}}`,
		"wrong page":       `{"success":true,"result":[],"result_info":{"page":2,"count":0,"total_count":0}}`,
		"count mismatch":   `{"success":true,"result":[],"result_info":{"count":1,"total_count":0}}`,
		"duplicate":        `{"success":true,"result":[{"id":"a"},{"id":"a"}],"result_info":{"count":2,"total_count":2}}`,
		"exceeds total":    `{"success":true,"result":[{"id":"a"}],"result_info":{"count":1,"total_count":0}}`,
		"bad id":           `{"success":true,"result":[{"id":"../a"}],"result_info":{"count":1,"total_count":1}}`,
		"failure":          `{"success":false,"errors":[{"message":"denied"}]}`,
		"malformed":        `{`,
	} {
		t.Run(name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer s.Close()
			_, err := NewClient(&Config{BaseURL: s.URL}).ListStatusPageOrder(StatusPageOrderScope{PageID: "page1234", Kind: "components"})
			if err == nil {
				t.Fatal("expected failure")
			}
		})
	}
	for _, changeTotal := range []bool{false, true} {
		t.Run(fmt.Sprint("pagination-change-", changeTotal), func(t *testing.T) {
			calls := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				total := 2
				if changeTotal && calls > 1 {
					total = 3
				}
				fmt.Fprintf(w, `{"success":true,"result":[{"id":"a"}],"result_info":{"count":1,"total_count":%d}}`, total)
			}))
			defer s.Close()
			_, err := NewClient(&Config{BaseURL: s.URL}).ListStatusPageOrder(StatusPageOrderScope{PageID: "page1234", Kind: "components"})
			if err == nil || calls != 2 {
				t.Fatalf("err %v calls %d", err, calls)
			}
		})
	}
}

func TestOrderPutContract(t *testing.T) {
	for _, kind := range []string{"components", "groups", "group_components"} {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprint(kind, empty), func(t *testing.T) {
				ids := []string{"second", "first"}
				if empty {
					ids = nil
				}
				scope := StatusPageOrderScope{PageID: "page1234", Kind: kind}
				endpoint := "/components/sort-order"
				key := "component_ids"
				if kind == "groups" {
					endpoint = "/groups/sort-order"
					key = "group_ids"
				}
				if kind == "group_components" {
					endpoint = "/groups/group123/sort-order"
					scope.GroupID = "group123"
				}
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "PUT" || r.URL.Path != "/v1/status_pages/page1234"+endpoint || r.Header.Get("Authorization") != "Bearer test" {
						t.Errorf("bad request %s %s", r.Method, r.URL)
					}
					var body map[string][]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					got, ok := body[key]
					if !ok || got == nil || len(body) != 1 || len(got) != len(ids) {
						t.Errorf("body %v", body)
					}
					if !empty && !reflect.DeepEqual(got, ids) {
						t.Errorf("order %v", got)
					}
					fmt.Fprint(w, `{"success":true,"result":{"message":"OK"}}`)
				}))
				defer s.Close()
				if err := NewClient(&Config{BaseURL: s.URL, APIKey: "test"}).SetStatusPageOrder(scope, ids); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestOrderPutErrorsAndPathSafety(t *testing.T) {
	for _, body := range []string{`{`, `{"success":false,"errors":[{"message":"denied"}]}`} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		err := NewClient(&Config{BaseURL: s.URL}).SetStatusPageOrder(StatusPageOrderScope{PageID: "page", Kind: "groups"}, []string{})
		s.Close()
		if err == nil {
			t.Fatal("expected error")
		}
	}
	for _, id := range []string{"", "..", "a/b", "a?x", "a#x", "%2f", "white space", "a\\b"} {
		if ValidOrderingID(id) {
			t.Errorf("unsafe %q", id)
		}
	}
	c := NewClient(&Config{BaseURL: "http://invalid.invalid"})
	for _, scope := range []StatusPageOrderScope{{PageID: "../bad", Kind: "groups"}, {PageID: "page", Kind: "group_components", GroupID: "bad/path"}, {PageID: "page", Kind: "unknown"}} {
		if err := c.SetStatusPageOrder(scope, nil); err == nil {
			t.Fatal("invalid scope allowed")
		}
	}
	for _, ids := range [][]string{{"a", "a"}, {""}, {"../a"}} {
		if err := c.SetStatusPageOrder(StatusPageOrderScope{PageID: "page", Kind: "groups"}, ids); err == nil {
			t.Fatal("invalid IDs allowed")
		}
	}
}

func TestHTTPNotFoundClassification(t *testing.T) {
	for _, status := range []int{401, 403, 404, 500} {
		for _, body := range []string{`{"errors":[{"code":404,"message":"not found"}]}`, `not found`} {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); fmt.Fprint(w, body) }))
			_, err := NewClient(&Config{BaseURL: s.URL}).Get("/test")
			s.Close()
			if IsNotFound(fmt.Errorf("wrapped: %w", err)) != (status == 404) {
				t.Fatalf("wrong classification %d %v", status, err)
			}
			var h *HTTPError
			if !errors.As(err, &h) || h.StatusCode != status {
				t.Fatalf("lost HTTP status: %v", err)
			}
		}
	}
	if IsNotFound(fmt.Errorf("404 not found")) {
		t.Fatal("matched message")
	}
}
