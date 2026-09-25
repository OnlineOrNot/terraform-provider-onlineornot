# Terraform Provider OnlineOrNot

**Generated:** 2026-09-25 | **Commit:** e6f2d4c | **Branch:** main

## Overview

Go Terraform Plugin Framework provider for OnlineOrNot. Hand-written API clients and resource lifecycles sit around generated base schemas; `provider.go` registers 17 resources and 7 data sources.

## Structure

```text
terraform-provider-onlineornot/
├── main.go                    # Provider server; go:generate delegates to make generate
├── schema.lock.json           # Upstream repository, revision, path, SHA-256
├── generator_config.yml       # OpenAPI operation mapping and schema overrides
├── operation-parity.json      # Reviewed disposition of every API operation
├── internal/
│   ├── client/                # HTTP contract; read its AGENTS.md for client changes
│   └── provider/              # Terraform lifecycle; read its AGENTS.md for state/schema changes
├── scripts/                   # Contract and example tooling; read its AGENTS.md before edits
├── templates/                 # Durable source for registry prose
├── examples/                  # Terraform configs and companion Playwright scripts
├── docs/                      # Generated registry documentation
└── tools/enrich-docs/          # Adds endpoint-specific enum values to generated docs
```

## Where to Look

| Task | Location | Important boundary |
|------|----------|--------------------|
| Provider authentication/registration | `internal/provider/provider.go` | `api_key` config falls back to `ONLINEORNOT_API_KEY`; `base_url` supports loopback tests |
| Add an API-backed capability | `operation-parity.json`, `internal/client/`, `internal/provider/` | Generated schemas alone do not implement CRUD or register resources |
| Change generated schema | `generator_config.yml`, pinned API contract | Base schemas are in `internal/provider/resource_*/`; runtime overrides live in resource implementations |
| Change registry examples/prose | `examples/`, `templates/`, resource schema descriptions | Run `make docs` to update `docs/` |
| Fix enum enrichment | `tools/enrich-docs/main.go` | `resourcePaths`/`dataSourcePaths`, extraction, then Markdown enrichment |
| Review contract drift | `scripts/`, `schema.lock.json`, `operation-parity.json` | Every operation needs an implemented/planned/waived mapping |
| CI or releases | `.github/workflows/`, `.goreleaser.yml`, `GNUmakefile` | Generation must leave tracked files unchanged in CI |

## Generation Boundaries

1. `make fetch-schema` downloads the revision in `schema.lock.json` and verifies its digest. The sibling `api-schemas/` checkout is not the input.
2. `scripts/prepare-openapi.py` projects supported success envelopes into `openapi.codegen.json` without modifying the pinned contract.
3. `make generate-schemas` produces `provider_code_spec.json`, then generated resource packages using `generator_config.yml`.
4. `make docs` formats examples, renders registry docs, then enriches enum descriptions.

- `openapi.json`, `openapi.codegen.json`, and `provider_code_spec.json` are ignored intermediates; generated Go schemas and registry docs are tracked.
- Never hand-edit `*_resource_gen.go`. Change the generator inputs or hand-written schema overrides, then regenerate and inspect the diff.
- Data sources are hand-written; the framework generation command generates resources only. Tokens, ordering, and typed checks also have hand-written schema behavior.
- `go generate ./...` runs the full `make generate` pipeline, including downloads and doc/example rewrites.
- Upstream sync copies the API SDK's schema lock daily and opens/updates a PR; classify newly introduced operations before considering the sync complete.

## Documentation Tooling

- Despite its name and log message, `fetchOpenAPISpec` in `tools/enrich-docs/main.go` reads local `openapi.json`; `make docs` performs the download first.
- Enrichment uses resource POST request schemas and data-source GET 200 response schemas. New endpoint coverage requires updating the path maps.
- Enum lookup tracks nested Markdown schema context; values are sorted, and existing `Must be one of:` text is retained. Regenerate docs before rerunning enrichment after enum changes.

## Verification

Run commands from this repository, using the Go version in `go.mod` (currently 1.25.5).

```bash
make build                 # Compile packages
TF_ACC= make test           # Python checks + Go tests, with live acceptance disabled
make check-examples        # Local provider build + Terraform init/validate; no API calls
make check-contract        # Fetch verified schema, then check operation coverage
make generate-schemas      # Rewrite generated resource schemas
make docs                  # Rewrite formatted examples and registry docs
make generate              # Full schema + docs pipeline
```

- Local lifecycle tests use Terraform against loopback HTTP servers; install Terraform or set `TF_ACC_TERRAFORM_PATH`. CI pins Terraform 1.13.5 for this suite.
- `make testacc` sets `TF_ACC=1` and mutates real API resources using `ONLINEORNOT_API_KEY`. CI runs live acceptance only on pushes to `main`.
- `make install` targets only version `0.0.1` on `darwin_arm64`. Its `build` prerequisite uses `go build ./...`; verify the root provider binary exists and is current before copying it.
- `make release VERSION=...` creates and pushes a tag; it is a release operation, not verification.
- `test-local/` is ignored and may contain credentials and Terraform state. Keep its contents out of documentation, logs, and commits.
