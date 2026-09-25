# Provider Package

Terraform schema overrides, lifecycle operations, import handling, and API/state conversion.

## Where to Look

| Task | Location | Boundary |
|------|----------|----------|
| Registration and shared client | `provider.go`: `Resources`, `DataSources`, `Configure` | Constructors must be registered explicitly |
| Generic/uptime/browser checks | `check_resource.go`: `CheckResource`, `checkModel` | Three constructors share behavior and select endpoint/type |
| DNS/TCP checks | `typed_check_resources.go`: `typedCheckModel`, `typedCheckSchema` | Hand-written typed models reuse generated assertion types |
| Omitted fields and defaults | `check_plan_modifiers.go`: `preserveCheckConfiguration` | Defaults on creation; prior state on omitted updates |
| PATCH field selection | `check_patch.go`: `configuredCheckPatch` | Select configured, fully known fields before serialization |
| Pause/mute transitions | `operational_state.go` | One state field per request; disable old state first |
| Status-page group relationships | `status_page_component_resource.go`, `status_page_component_group_resource.go` | Relationship patches differ from ordinary scalar updates |
| Complete collection ordering | `status_page_order_resource.go` | One implementation, three scope-specific constructors |
| Replacement-only credentials | `token_resource.go` | Hand-written schema, secret preservation, redacted diagnostics |
| Data-source reads | `*_data_source.go` | Hand-written schemas and result conversion |
| Generated base models/types | `resource_*/` | Runtime schema and models can differ from these bases |

## Schema and State Rules

- Inspect the resource's `Schema` method before changing generated inputs: it may override descriptions, defaults, sensitivity, attributes, and plan modifiers.
- A new field needs coverage in the schema/model, client wire type, Create request, Update request/PATCH selection, and Read/API-to-state conversion. Check all relevant monitor variants and data sources.
- After Create/Update, resolve unknown state values to concrete values or correctly typed nulls. When filling unresolved fields, guard with `IsUnknown()` so configured values survive.
- Nested list nulls require the generated custom element type and its `AttributeTypes(ctx)`; a plain object type can mismatch the schema.
- Use prior state for request identity when the plan ID is unknown. `check_patch_test.go` covers the component-update regression.
- Import formats are resource-specific: nested status-page objects use parent/child IDs; ordering uses page ID or page/group ID. Validate components before indexing them.

## Monitor Semantics

- `preserveCheckConfiguration` keeps omitted optional/computed values, including prior nulls, on updates. Derived fields such as check timeout/version need their own plan logic.
- `configuredCheckPatch` reads raw config and JSON tags; unknown or null fields must not become zero-valued updates. Explicit empty lists/maps are clears.
- `paused` and `muted` cannot both be true. Use `operationalStateChanges` and `applyOperationalState` to order single-field API transitions.
- Scripted browser checks require timeout to be omitted; URL-based checks default to 10000 ms. Script transitions may clear URL and recompute runtime version.
- When editing shared monitor fields, inspect `monitor_field_parity_test.go`, `omitted_check_plans_test.go`, `defaults_lifecycle_test.go`, and script lifecycle coverage.

## Special Lifecycles

- Ordering owns the complete ordered list for one scope. Check parent existence and exact membership before PUT; save ownership before verification so failures remain recoverable.
- Ordering Delete intentionally makes no API change. It relinquishes ownership without deleting members or resetting ranks.
- Tokens have no API update: configuration changes require replacement. Preserve the creation-only secret on refresh; imported tokens have null secrets.
- Keep token creation intent (`expires_at`, `never_expires`) separate from returned `expires_after`; equivalent timestamps should not cause replacement churn.
- Token diagnostics go through `tokenError`; raw API errors may echo secrets.

## Verification Paths

- `provider_test.go` supplies protocol-v6 factories and the API-key precheck for live acceptance tests.
- Many `*_test.go` files are local tests, not live acceptance. Loopback lifecycle tests use dummy credentials and explicit `base_url`; inspect the test setup before selecting a run.
- `check_script_file_test.go` exercises Terraform `file()` and `fileset()` behavior. Its CLI path handles `for_each` string keys that the legacy plugin-testing state shim cannot represent.
- `status_page_order_cli_test.go` verifies ordering through Terraform CLI; the companion resource tests cover lifecycle failures and state handling.
- Run `TF_ACC= go test ./internal/provider` from the root for local provider verification. Keep loopback tests isolated from provider tokens and remote backends; retain repeatable plan/state evidence when adding E2E coverage.
