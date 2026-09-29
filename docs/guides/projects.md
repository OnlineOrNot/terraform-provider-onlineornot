---
page_title: "Projects"
description: |-
  Select projects and move monitors without replacing their identity.
---

# Projects

## Availability

Project support requires an API deployment implementing the projects contract. The provider's project fixtures run locally and do not establish production availability. Before publishing this provider, verify the released API schema and SDK, update the pinned schema and operation coverage, regenerate, and record the provider release version. The provider uses its own Go HTTP client rather than importing the SDK.

## Selection and compatibility

Use `onlineornot_project` to create or rename a project and `data.onlineornot_projects` to list projects. Project management requires `PROJECTS:EDIT` (`READ` suffices for listing); this grants no check, heartbeat or secret permissions.

`project_id` selects ownership for checks, heartbeats and environment variables. Use the encoded ID, not a name or raw database ID. Names are editable and need not be unique. Import a project using its encoded ID. `is_default` reports stable Default identity independently of its name.

Omitting `project_id` on creation selects Default server-side. Omitting it on an existing resource retains ownership, rather than moving it back to Default. The checks and heartbeats data sources accept an optional `project_id` filter; omission keeps organisation-wide authorized listing. Returned entries include their owning project ID. Status pages and notification integrations remain organisation-scoped.

```terraform
resource "onlineornot_project" "staging" {
  name = "staging"
}

resource "onlineornot_uptime_check" "staging" {
  name       = "Staging homepage"
  url        = "https://staging.example.com"
  project_id = onlineornot_project.staging.id
}

data "onlineornot_checks" "staging" {
  project_id = onlineornot_project.staging.id
}
```

## Ownership updates

Changing a monitor's `project_id` uses `POST /v1/checks/{id}/move` or `POST /v1/heartbeats/{id}/move`, never an ownership PATCH or replacement. The move request contains only the destination project. The API must preserve identity, history, ping URLs, scheduling state and pause/disable reasons, and atomically resolve referenced variable names and types in the destination. Destination variables must already exist; variables are not copied and there is no fallback to Default.

A rejected move prevents subsequent configuration updates. A successful move is recorded in Terraform state before any remaining configuration writes. A move and other configuration changes are separate API operations, not a single transaction: if a later update fails, ownership remains in the destination. Refresh and retry after resolving the error.

The move itself does not reactivate a monitor. Explicit changes to `paused` or `muted` in the same plan remain separate requested changes. Already-dispatched work may finish with its original snapshot; work selected after commit must use the destination, and racing selection must observe a coherent snapshot. Terraform does not cancel in-flight work or enforce the server's execution boundary.

Variables are project-local. The same name may exist in different projects, but only once within each project. Variable moves are unsupported: changing a variable's project requires replacement and may fail while existing checks reference it. Prepare destination variables before moving checks; use explicit dependencies where necessary.

Only empty non-Default projects can be deleted. The API enforces these rules, including concurrency protection. Destroy dependent resources first; importing Default does not make it deletable.

## Local verification and release gate

Run with Terraform installed locally (or set `TF_ACC_TERRAFORM_PATH`):

```shell
TF_ACC= go test ./internal/client ./internal/provider
```

`TestProjectContract`, `TestProjectFiltersAcrossPages`, `TestProjectMoveAndDeleteFailures`, `TestProjectCreateSelectionWireModels`, `TestProjectSelectionSchemas`, `TestProjectMoveFailurePreservesState`, `TestProjectCRUDLifecycle`, `TestProjectDataSourcesLifecycle` and `TestProjectOwnershipLifecycle` use synthetic IDs, dummy credentials and loopback HTTP fixtures. They cover CRUD/import, encoded IDs, omitted creation and list compatibility, pagination filters, move-only ownership payloads, rejection handling and identity/state preservation for all monitor variants. No live API resources are needed.

These fixtures do **not** prove server authorization, variable isolation, atomic reference rebinding, migration/backfill, concurrent deletion, scheduling races, dispatch snapshots or real ping/history continuity. Those require API-side evidence. Project publication remains blocked until the released contract is reconciled and a human records the provider release version/link; local fixture success is not a release claim.

### Fixture verification report

| Verification | Result |
| --- | --- |
| Initial contract/schema tests before implementation | Failed: missing project methods and selection attributes |
| Ownership lifecycle before heartbeat fix | Failed: unknown heartbeat ID and omitted values after update |
| Project CRUD/import and filtered data sources | Passed against loopback fixtures |
| Ownership lifecycle matrix | Passed: six monitor resource variants × active/paused/disabled fixtures |
| Client rejection and pagination tests | Passed; no live credentials |
| Full `TF_ACC= make test` and `make build` | Passed using a locally installed Terraform CLI |
| `make check-examples` | Passed: 15 configurations, local provider mirror only |
| Pinned operation coverage | Passed: 98 existing operations; not evidence of candidate project coverage |

Terraform's automatic installer initially failed its checksum-signature verification because of an expired signing key. Verification used an existing local binary through `TF_ACC_TERRAFORM_PATH`; no remote acceptance tests were enabled. Documentation was generated and enriched using the unchanged, digest-verified pinned OpenAPI contract. The project schema lock and candidate operation entries remain a release dependency, not an implicit update to the released contract.
