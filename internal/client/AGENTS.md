# API Client Package

HTTP client for OnlineOrNot API. All provider CRUD operations use this client.

## Structure

```
client/
├── client.go          # Base client, Config, HTTP helpers
├── client_test.go     # Unit tests with mock HTTP server
├── checks.go          # Check CRUD
├── heartbeats.go      # Heartbeat CRUD
├── maintenance_windows.go
├── status_pages.go
├── status_page_components.go
├── status_page_component_groups.go
├── status_page_incidents.go
├── status_page_scheduled_maintenances.go
├── users.go
└── webhooks.go
```

## Where to Look

| Task | File |
|------|------|
| Add new API method | Create `<resource>.go` or add to existing |
| Modify request/response | `client.go` for helpers, specific file for structs |
| Add unit tests | `client_test.go` |
| Check API response format | `client.go` -> `APIResponse[T]` |

## Patterns

### API Response Wrapper

```go
type APIResponse[T any] struct {
    Result   T            `json:"result"`
    Success  bool         `json:"success"`
    Errors   []APIError   `json:"errors"`
    Messages []APIMessage `json:"messages"`
}
```

### HTTP Request Pattern

```go
func (c *Client) GetCheck(id string) (*Check, error) {
    var resp APIResponse[Check]
    err := c.doRequest("GET", fmt.Sprintf("/checks/%s", id), nil, &resp)
    if err != nil {
        return nil, err
    }
    return &resp.Result, nil
}
```

### Struct Naming

- Request/Response structs in same file as method
- JSON tags use snake_case: `json:"test_interval,omitempty"`
- Optional booleans use pointer: `FollowRedirects *bool`

### Unit Test Pattern

```go
func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
    server := httptest.NewServer(handler)
    client := NewClient(&Config{
        APIKey:  "test-api-key",
        BaseURL: server.URL,
    })
    return server, client
}

func TestClient_GetCheck(t *testing.T) {
    server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.Method != "GET" { t.Errorf(...) }
        if r.URL.Path != "/checks/123" { t.Errorf(...) }
        // Return mock response
        json.NewEncoder(w).Encode(APIResponse[Check]{...})
    })
    defer server.Close()
    // Test client method
}
```

## Anti-Patterns

### Omit Empty

Always use `omitempty` for optional fields to avoid sending null/zero values:

```go
// CORRECT
TestInterval int `json:"test_interval,omitempty"`

// WRONG - sends 0 for unset fields
TestInterval int `json:"test_interval"`
```

### Pointer for Optional Bool

Required because Go's zero value for bool is false, which is meaningful:

```go
// CORRECT - nil means "not set"
FollowRedirects *bool `json:"follow_redirects,omitempty"`

// WRONG - false means "explicitly disabled" but also "not set"
FollowRedirects bool `json:"follow_redirects,omitempty"`
```
