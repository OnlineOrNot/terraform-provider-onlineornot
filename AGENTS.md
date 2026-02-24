# Terraform Provider OnlineOrNot

**Generated:** 2026-02-24 | **Commit:** 9d72d6a | **Branch:** main

## Overview

Terraform provider for OnlineOrNot uptime monitoring. Go + terraform-plugin-framework.

**Scale**: 39 Go files | 6.4k lines | 9 resources | 7 data sources

## Structure

```
terraform-provider-onlineornot/
├── main.go                    # Entry point (providerserver.Serve)
├── generator_config.yml       # Code generation config for schemas
├── internal/
│   ├── client/                # HTTP API client (11 files)
│   └── provider/              # Resources + Data Sources <- AGENTS.md
└── test-local/                # Local dev testing (gitignored)
```

## Where to Look

| Task                | Location                                                                                       | Notes |
| ------------------- | ---------------------------------------------------------------------------------------------- | ----- |
| Add new resource    | `generator_config.yml` → generate → `internal/provider/*_resource.go` + `internal/client/*.go` |
| Add new data source | Same pattern, under `data_sources:` in config                                                  |
| API client methods  | `internal/client/client.go` (base), `internal/client/*.go` (per-resource)                      |
| Provider config     | `internal/provider/provider.go`                                                                |
| Register resources  | `provider.go` → `Resources()` and `DataSources()` methods                                      |
| Schema definitions  | `internal/provider/resource_*/` (GENERATED - do not edit)                                      |

## Code Conventions

### API Response Wrapper

All API responses use Cloudflare-style wrapper:

```go
type APIResponse[T any] struct {
    Result   T            `json:"result"`
    Success  bool         `json:"success"`
    Errors   []APIError   `json:"errors"`
    Messages []APIMessage `json:"messages"`
}
```

### JSON Field Names

Use snake_case for API fields: `json:"test_interval,omitempty"`

### Optional Booleans

Use pointer for optional bools: `FollowRedirects *bool`

### Environment Variable

API key via `ONLINEORNOT_API_KEY` or provider config

## Anti-Patterns (THIS PROJECT)

### DO NOT EDIT Generated Files

All `*_resource_gen.go` files in `resource_*/` subdirectories are generated:

- `resource_check/check_resource_gen.go`
- `resource_heartbeat/heartbeat_resource_gen.go`
- etc.

Regenerate via code generator, not manual edits.

### Unknown Value Handling

After Create, MUST set unknown fields to null to avoid "unknown after apply" errors:

```go
if data.TestInterval.IsUnknown() {
    data.TestInterval = types.Int64Null()
}
```

Do NOT unconditionally set to null - check `IsUnknown()` first to preserve user-provided values.

## Commands

```bash
# Build
go build -o terraform-provider-onlineornot

# Local testing (requires ~/.terraformrc dev_overrides)
export ONLINEORNOT_API_KEY="your-key"
cd test-local && terraform plan

# Acceptance tests
make testacc
```

## Notes

- No `examples/` or `docs/` directories yet (removed scaffolding)
- Schemas generated from OpenAPI via `generator_config.yml` + external tool
- Status page resources use nested paths: `status_page_id/component_id` for imports
