# API Client Package

Hand-written HTTP transport and wire models used by provider resources and data sources.

## Where to Look

| Task | Location |
|------|----------|
| Authentication, HTTP verbs, status errors | `client.go`: `NewClient`, `doRequest`, `HTTPError`, `IsNotFound` |
| Generic/uptime/browser checks | `checks.go`: `Check`, `CheckPatch`, `checkRequest`, typed endpoint methods |
| DNS/TCP checks | `typed_checks.go`: wire models, patches, `parseAPIResponse` |
| Paginated collection reads | `pagination.go`: `listAll` |
| Status-page ordering | `statuspageordering.go`: `StatusPageOrderScope`, strict membership reads and PUTs |
| Component relationships | `statuspagecomponents.go`, `statuspagecomponentgroups.go` |
| Incident/maintenance payloads | `statuspageincidents.go`, `statuspagesmaintenance.go` |
| Other collection clients | `heartbeats.go`, `maintenancewindows.go`, `statuspages.go`, `users.go`, `webhooks.go` |
| Token create/read/delete contract | `tokens.go`: expiry encoding and secret-safe parsing |
| Wire-format regression coverage | `*_test.go`, especially `monitor_field_parity_test.go`, `status_page_contract_test.go`, `pagination_test.go` |

## Transport and Responses

- `BaseURL` is the origin; resource methods include `/v1` in their paths. Transport adds bearer authentication and a 30-second HTTP timeout.
- `Get`, `Post`, `Patch`, `Put`, and `Delete` return raw bytes. Decode the envelope in the resource method or use `parseAPIResponse[T]`; `doRequest` takes three arguments, not a response destination.
- HTTP success does not imply API success: JSON envelopes carry `success`, `result`, and `errors`. Preserve envelope checks, including acknowledgement checks for writes without a returned resource.
- Use `IsNotFound` for HTTP 404 detection; it examines `HTTPError.StatusCode`, independently of API error codes or text.
- Match each endpoint's actual verb and path. For example, status-page update uses POST and maintenance windows use `/v1/maintenance-windows`.

## Presence Is Part of the Contract

- Most fields use snake_case; token dates deliberately use camelCase (`expiresAt`, `expiresAfter`).
- Optional booleans use pointers when explicit `false` differs from omission. Nullable strings likewise need presence-aware representations.
- Do not apply `omitempty` indiscriminately: zero reminder/confirmation/recovery periods are meaningful and intentionally serialize.
- Check PATCH models carry a `Fields` map selected from Terraform configuration. Preserve explicit zero, false, empty strings, and empty collections; omitted fields remain unmanaged.
- `checkRequest` represents missing URL as JSON null, allowing a URL-based check to become script-controlled. Keep the response model's string URL separate from request encoding.
- `CreateTokenRequest.ExpiresAt` uses `**string`: nil means API default, pointer-to-nil means no expiry, pointer-to-value means explicit expiry.
- Token decoding uses generic errors because malformed responses can contain the secret. Token DELETE also checks `deleted` and treats HTTP 404 as already absent.

## Pagination and Ordering

- Ordinary list endpoints use `listAll[T]`, preserving existing query parameters and requesting numbered pages with `per_page=100`.
- Validate pagination metadata rather than assuming the requested page size is honored. `json.Number` accommodates numeric and quoted numeric page values.
- Ordering uses separate, stricter pagination: stable totals, unique path-safe IDs, correct page/count metadata, and forward progress are required before rank updates.
- Ordering reads preserve API order and filter ungrouped/within-group membership after reading the full component collection. Sorting IDs locally changes the meaning.
- Parent existence checks distinguish a missing page/group from an empty scope; ordering PUTs change ranks, not membership.

## Verification

Existing client tests use `httptest` servers to assert verbs, paths, bodies, and response/error handling. Run `go test ./internal/client` from the repository root for client-only changes; lifecycle/state verification belongs in the provider suite.
