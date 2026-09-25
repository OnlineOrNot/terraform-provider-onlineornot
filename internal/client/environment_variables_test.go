package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnvironmentVariableClientMetadataAndError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var input EnvironmentVariableWrite
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Name != "API_TOKEN" || input.Type != "secret" || input.Value == nil || *input.Value != "example-secret" {
				t.Error("invalid request")
			}
			fmt.Fprint(w, `{"success":true,"result":{"id":"abc","name":"API_TOKEN","type":"secret","has_value":true}}`)
			return
		}
		if r.Method == "PATCH" {
			w.WriteHeader(400)
			fmt.Fprint(w, `{"success":false,"errors":[{"message":"example-secret"}]}`)
			return
		}
		t.Error(r.Method)
	}))
	defer server.Close()
	c := NewClient(&Config{BaseURL: server.URL, APIKey: "test"})
	secret := "example-secret"
	result, err := c.CreateEnvironmentVariable(EnvironmentVariableWrite{Name: "API_TOKEN", Type: "secret", Value: &secret})
	if err != nil || result.ID != "abc" || strings.Contains(fmt.Sprintf("%+v", result), secret) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	_, err = c.UpdateEnvironmentVariable("abc", EnvironmentVariableWrite{Value: &secret})
	if err == nil {
		t.Fatal("expected error")
	}
	// The provider must discard API errors since the HTTP helper includes server messages.
	if !strings.Contains(err.Error(), secret) {
		t.Fatal("test fixture did not exercise secret echo")
	}
}
