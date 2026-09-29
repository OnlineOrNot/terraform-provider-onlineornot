package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectContract(t *testing.T) {
	const id = "a1b2c3d4e5f6g7h8"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Method == "PATCH" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if r.URL.Path == "/v1/checks/check-id/move" || r.URL.Path == "/v1/heartbeats/heartbeat-id/move" {
				if len(body) != 1 || body["project_id"] != id {
					t.Errorf("move must contain only destination: %v", body)
				}
			} else if len(body) != 1 || body["name"] != "staging" {
				t.Errorf("project write: %v", body)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "DELETE" {
			w.Write([]byte(`{"success":true,"result":{"id":"` + id + `"}}`))
			return
		}
		if r.URL.Path == "/v1/checks" {
			if r.URL.Query().Get("project_id") != id {
				t.Error("missing filter")
			}
			w.Write([]byte(`{"success":true,"result":[],"result_info":{"total_count":0}}`))
			return
		}
		responseID := id
		if r.URL.Path == "/v1/checks/check-id/move" {
			responseID = "check-id"
		}
		if r.URL.Path == "/v1/heartbeats/heartbeat-id/move" {
			responseID = "heartbeat-id"
		}
		w.Write([]byte(`{"success":true,"result":{"id":"` + responseID + `","name":"staging","is_default":false,"project_id":"` + id + `"}}`))
	}))
	defer server.Close()
	c := NewClient(&Config{APIKey: "dummy"})
	c.BaseURL = server.URL
	if _, err := c.CreateProject("staging"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetProject(id); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateProject(id, "staging"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteProject(id); err != nil {
		t.Fatal(err)
	}
	if err := c.MoveCheck("check-id", id); err != nil {
		t.Fatal(err)
	}
	if err := c.MoveHeartbeat("heartbeat-id", id); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListChecksInProject(id); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidProjectIDsNeverSend(t *testing.T) {
	c := NewClient(&Config{APIKey: "dummy"})
	c.BaseURL = "http://127.0.0.1:1"
	for _, id := range []string{"1", "Default", "../projects", "1234567890123456"} {
		if ValidProjectID(id) {
			t.Errorf("accepted %q", id)
		}
		if _, err := c.GetProject(id); err == nil {
			t.Errorf("accepted %q", id)
		}
	}
}

func TestProjectMoveAndDeleteFailures(t *testing.T) {
	for _, status := range []int{400, 403, 404, 409, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				w.Write([]byte(`{"success":false,"errors":[{"message":"fixture rejection"}]}`))
			}))
			defer server.Close()
			c := NewClient(&Config{APIKey: "dummy", BaseURL: server.URL})
			if err := c.MoveCheck("check12345678", "projectFixture1234"); err == nil {
				t.Error("move rejection ignored")
			}
			if err := c.MoveHeartbeat("heartbeat12345678", "projectFixture1234"); err == nil {
				t.Error("move rejection ignored")
			}
			err := c.DeleteProject("projectFixture1234")
			if (err == nil) != (status == 404) {
				t.Errorf("delete status %d: %v", status, err)
			}
		})
	}
	for _, body := range []string{`{"success":false}`, `{"success":true,"result":{"id":"wrong","project_id":"projectFixture1234"}}`, `{"success":true,"result":{"id":"check12345678","project_id":"wrong"}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		c := NewClient(&Config{APIKey: "dummy", BaseURL: server.URL})
		if err := c.MoveCheck("check12345678", "projectFixture1234"); err == nil {
			t.Error("invalid acknowledgement accepted")
		}
		server.Close()
	}
}

func TestProjectCreateSelectionWireModels(t *testing.T) {
	for _, project := range []string{"", "projectFixture1234"} {
		for _, input := range []any{&Check{ProjectID: project}, &DNSCheck{ProjectID: project}, &TCPCheck{ProjectID: project}, &Heartbeat{ProjectID: project}, EnvironmentVariableWrite{ProjectID: project}} {
			body, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]any
			json.Unmarshal(body, &fields)
			value, present := fields["project_id"]
			if project == "" && present || project != "" && value != project {
				t.Errorf("%T selection mismatch: %s", input, body)
			}
		}
	}
}

func TestProjectFiltersAcrossPages(t *testing.T) {
	for _, collection := range []string{"checks", "heartbeats"} {
		for _, project := range []string{"", "projectFixture1234"} {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/v1/"+collection || r.URL.Query().Get("project_id") != project {
					t.Errorf("unexpected list URL %s", r.URL)
				}
				if project == "" && r.URL.Query().Has("project_id") {
					t.Error("omitted filter must stay absent")
				}
				fmt.Fprintf(w, `{"success":true,"result":[{"id":"item%d"}],"result_info":{"total_count":2,"page":%d,"count":1}}`, calls, calls)
			}))
			c := NewClient(&Config{APIKey: "dummy", BaseURL: server.URL})
			var err error
			if collection == "checks" {
				_, err = c.ListChecksInProject(project)
			} else {
				_, err = c.ListHeartbeatsInProject(project)
			}
			if err != nil || calls != 2 {
				t.Errorf("pagination calls=%d err=%v", calls, err)
			}
			server.Close()
		}
	}
}
