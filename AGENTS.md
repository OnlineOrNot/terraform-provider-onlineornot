# Terraform Provider OnlineOrNot

**Generated:** 2026-03-06 | **Commit:** 06d183a | **Branch:** main

## Overview

Terraform provider for OnlineOrNot uptime monitoring. Go + terraform-plugin-framework.

**Scale**: 39 Go files | 7.5k lines | 9 resources | 7 data sources

## Structure

```
terraform-provider-onlineornot/
├── main.go                    # Entry point (providerserver.Serve)
├── generator_config.yml       # Code generation config for schemas
├── internal/
│   ├── client/                # HTTP API client <- AGENTS.md
│   └── provider/              # Resources + Data Sources <- AGENTS.md
├── docs/                      # Terraform registry docs (generated)
├── examples/                  # Example TF configs
├── templates/                 # Doc generation templates
├── tools/enrich-docs/         # Doc enrichment tool
└── test-local/                # Local dev testing (gitignored)
```

## Where to Look

| Task | Location | Notes |
|------|----------|-------|
| Add new resource | `generator_config.yml` -> generate -> `internal/provider/*_resource.go` + `internal/client/*.go` | |
| Add new data source | Same pattern, under `data_sources:` in config | |
| API client methods | `internal/client/client.go` (base), `internal/client/*.go` (per-resource) | |
| Provider config | `internal/provider/provider.go` | |
| Register resources | `provider.go` -> `Resources()` and `DataSources()` methods | |
| Schema definitions | `internal/provider/resource_*/` | GENERATED - do not edit |
| Acceptance tests | `internal/provider/*_test.go` | Requires `TF_ACC=1` |
| Unit tests | `internal/client/client_test.go` | Mock HTTP server |

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

Use snake_case: `json:"test_interval,omitempty"`

### Optional Booleans

Use pointer: `FollowRedirects *bool`

### Environment Variable

API key via `ONLINEORNOT_API_KEY` or provider config

## Adding a New Field to a Resource

When adding a new field, update **all 4 locations** or Terraform will error with "unknown value after apply":

1. **Client struct** (`internal/client/*.go`)
   ```go
   type Check struct {
       // ...
       Script string `json:"script,omitempty"`
   }
   ```

2. **Create function** (`internal/provider/*_resource.go`)
   ```go
   check := &client.Check{
       // ...
       Script: data.Script.ValueString(),
   }
   ```

3. **Update function** (same file)
   ```go
   check := &client.Check{
       // ...
       Script: data.Script.ValueString(),
   }
   ```

4. **populateModelFromAPI** (same file) - set to value OR null, never leave unknown:
   ```go
   if check.Script != "" {
       data.Script = types.StringValue(check.Script)
   } else {
       data.Script = types.StringNull()
   }
   ```

Missing any of these causes: `Provider returned invalid result object after apply... unknown value for X`

## Anti-Patterns (THIS PROJECT)

### DO NOT EDIT Generated Files

All `*_resource_gen.go` files in `resource_*/` subdirectories are generated. Regenerate via `go generate ./...`, not manual edits.

### Unknown Value Handling

After Create, MUST check `IsUnknown()` before setting null:

```go
// CORRECT
if data.TestInterval.IsUnknown() {
    data.TestInterval = types.Int64Null()
}

// WRONG - overwrites user values
data.TestInterval = types.Int64Null()
```

### Complex List Null Types

For nested object lists, construct the element type explicitly:

```go
if data.Components.IsUnknown() {
    elemType := resource_status_page_incident.ComponentsType{
        ObjectType: types.ObjectType{
            AttrTypes: resource_status_page_incident.ComponentsValue{}.AttributeTypes(ctx),
        },
    }
    data.Components = types.ListNull(elemType)
}
```

### Hardcoded darwin_arm64

`make install` hardcodes darwin_arm64 - won't work on Linux or Intel Macs.

## Commands

```bash
# Build
make build                  # or: go build -v ./...

# Unit tests
make test                   # or: go test -v ./...

# Acceptance tests (requires ONLINEORNOT_API_KEY)
make testacc                # or: TF_ACC=1 go test ./... -v -timeout 120m

# Generate docs
make docs                   # terraform-plugin-docs + enrich-docs tool

# Local install (darwin_arm64 only)
make install

# Release (creates and pushes version tag)
make release VERSION=0.2.0
```

## Notes

- **CI**: Acceptance tests only run on push to main (not PRs) to protect API key
- **Daily sync**: `check-upstream.yml` workflow auto-creates PR if upstream schemas change
- **Nested imports**: Status page resources use `status_page_id/component_id` format
- **Doc enrichment**: `tools/enrich-docs/` fetches OpenAPI spec to add enum values to docs
