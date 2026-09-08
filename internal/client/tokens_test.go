package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestTokenClientContract(t *testing.T) {
	for _, expiry := range []*string{nil, tokenString("2030-01-01T00:00:00Z")} {
		t.Run(fmt.Sprint(expiry != nil), func(t *testing.T) {
			server, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-api-key" {
					t.Error("missing authentication")
				}
				switch r.Method {
				case "POST":
					if r.URL.Path != "/v1/tokens" {
						t.Error(r.URL.Path)
					}
					var input map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						t.Fatal(err)
					}
					expected, _ := json.Marshal(expiry)
					if string(input["expiresAt"]) != string(expected) {
						t.Errorf("expiry: %s", input["expiresAt"])
					}
					if string(input["name"]) != `"example"` || string(input["grants"]) != `[{"scope":"UPTIME_CHECKS","permission":"READ"}]` || len(input) != 3 {
						t.Errorf("unexpected input: %v", input)
					}
					fmt.Fprint(w, `{"success":true,"result":{"id":"token123","name":"example","token":"mock-secret","expiresAfter":null}}`)
				case "GET":
					if r.URL.EscapedPath() != "/v1/tokens/token%2F123" {
						t.Error(r.URL.EscapedPath())
					}
					fmt.Fprint(w, `{"success":true,"result":{"id":"token123","name":"example","expiresAfter":null,"grants":[{"scope":"UPTIME_CHECKS","permission":"READ"}]}}`)
				case "DELETE":
					if r.URL.EscapedPath() != "/v1/tokens/token%2F123" {
						t.Error(r.URL.EscapedPath())
					}
					fmt.Fprint(w, `{"success":true,"result":{"deleted":true}}`)
				default:
					t.Errorf("unexpected method %s", r.Method)
				}
			})
			defer server.Close()
			token, err := c.CreateToken(&CreateTokenRequest{Name: "example", Grants: []TokenGrant{{Scope: "UPTIME_CHECKS", Permission: "READ"}}, ExpiresAt: &expiry})
			if err != nil || token.Token != "mock-secret" {
				t.Fatalf("create: %v", err)
			}
			token, err = c.GetToken("token/123")
			if err != nil || token.Token != "" || token.ExpiresAfter != nil || len(token.Grants) != 1 {
				t.Fatalf("read: %v", err)
			}
			if err := c.DeleteToken("token/123"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func tokenString(s string) *string { return &s }

func TestTokenClientErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		body     string
		notFound bool
	}{
		{"structured404", 404, `{"success":false,"errors":[{"code":10006,"message":"access denied"}]}`, true},
		{"plain404", 404, `not found`, true},
		{"forbidden", 403, `{"success":false,"errors":[{"code":10006,"message":"not found 404"}]}`, false},
		{"unauthorized", 401, `unauthorized`, false},
		{"server", 500, `server error`, false},
		{"malformed", 200, `{"mock-secret":`, false},
		{"unsuccessful", 200, `{"success":false,"result":{"id":"id","token":"mock-secret"}}`, false},
		{"emptyResult", 200, `{"success":true,"result":null}`, false},
		{"missingSecret", 200, `{"success":true,"result":{"id":"id"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, c := newTestServer(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) })
			defer server.Close()
			_, err := c.CreateToken(&CreateTokenRequest{})
			if err == nil || IsNotFound(fmt.Errorf("wrapped: %w", err)) != tc.notFound {
				t.Fatalf("create error: %v", err)
			}
			_, err = c.GetToken("id")
			if tc.name != "missingSecret" && (err == nil || IsNotFound(err) != tc.notFound) {
				t.Fatalf("get error: %v", err)
			}
			err = c.DeleteToken("id")
			if (err == nil) != tc.notFound {
				t.Fatalf("delete error: %v", err)
			}
		})
	}
}

func TestCreateTokenDefaultExpiry(t *testing.T) {
	server, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["expiresAt"]; ok {
			t.Error("default must omit expiresAt")
		}
		fmt.Fprint(w, `{"success":true,"result":{"id":"id123","token":"mock-secret","expiresAfter":"2030-01-01T00:00:00.000Z"}}`)
	})
	defer server.Close()
	token, err := c.CreateToken(&CreateTokenRequest{Name: "default", Grants: []TokenGrant{}})
	if err != nil || token.ExpiresAfter == nil {
		t.Fatalf("create default: %v", err)
	}
}
