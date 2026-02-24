package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer creates a mock HTTP server for testing
func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	server := httptest.NewServer(handler)
	client := NewClient(&Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	return server, client
}

func TestClient_GetCheck(t *testing.T) {
	check := Check{
		ID:            "abc123",
		Name:          "Test Check",
		URL:           "https://example.com",
		CheckType:     "UPTIME",
		Method:        "GET",
		TestInterval:  180,
		Timeout:       10000,
		AlertPriority: "HIGH",
	}

	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/checks/abc123" {
			t.Errorf("expected /v1/checks/abc123, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Bearer auth header")
		}

		// Return mock response
		resp := APIResponse[Check]{
			Result:  check,
			Success: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.GetCheck("abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != check.ID {
		t.Errorf("expected ID %s, got %s", check.ID, result.ID)
	}
	if result.Name != check.Name {
		t.Errorf("expected Name %s, got %s", check.Name, result.Name)
	}
	if result.URL != check.URL {
		t.Errorf("expected URL %s, got %s", check.URL, result.URL)
	}
}

func TestClient_CreateCheck(t *testing.T) {
	input := &Check{
		Name: "New Check",
		URL:  "https://example.com",
	}

	createdCheck := Check{
		ID:            "xyz789",
		Name:          "New Check",
		URL:           "https://example.com",
		CheckType:     "UPTIME",
		Method:        "GET",
		TestInterval:  180,
		Timeout:       10000,
		AlertPriority: "HIGH",
	}

	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/checks" {
			t.Errorf("expected /v1/checks, got %s", r.URL.Path)
		}

		// Decode and verify request body
		var reqBody Check
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.Name != input.Name {
			t.Errorf("expected Name %s, got %s", input.Name, reqBody.Name)
		}

		resp := APIResponse[Check]{
			Result:  createdCheck,
			Success: true,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.CreateCheck(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != createdCheck.ID {
		t.Errorf("expected ID %s, got %s", createdCheck.ID, result.ID)
	}
}

func TestClient_UpdateCheck(t *testing.T) {
	input := &Check{
		Name: "Updated Check",
	}

	updatedCheck := Check{
		ID:        "abc123",
		Name:      "Updated Check",
		URL:       "https://example.com",
		CheckType: "UPTIME",
	}

	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v1/checks/abc123" {
			t.Errorf("expected /v1/checks/abc123, got %s", r.URL.Path)
		}

		resp := APIResponse[Check]{
			Result:  updatedCheck,
			Success: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.UpdateCheck("abc123", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != updatedCheck.Name {
		t.Errorf("expected Name %s, got %s", updatedCheck.Name, result.Name)
	}
}

func TestClient_DeleteCheck(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v1/checks/abc123" {
			t.Errorf("expected /v1/checks/abc123, got %s", r.URL.Path)
		}

		resp := APIResponse[struct{ ID string }]{
			Result:  struct{ ID string }{ID: "abc123"},
			Success: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	err := client.DeleteCheck("abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_ListChecks(t *testing.T) {
	checks := []Check{
		{ID: "check1", Name: "Check 1", URL: "https://example1.com"},
		{ID: "check2", Name: "Check 2", URL: "https://example2.com"},
	}

	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/checks" {
			t.Errorf("expected /v1/checks, got %s", r.URL.Path)
		}

		resp := APIListResponse[Check]{
			Result:  checks,
			Success: true,
			ResultInfo: ResultInfo{
				Page:       1,
				PerPage:    20,
				Count:      2,
				TotalCount: 2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.ListChecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 checks, got %d", len(result))
	}
}

func TestClient_APIError(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse[Check]{
			Success: false,
			Errors: []APIError{
				{Code: 1001, Message: "Check not found"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	_, err := client.GetCheck("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMsg := "API error: Check not found (code: 1001)"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}
