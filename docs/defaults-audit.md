# Resource default audit

Audited against onlineornot-next commit `d220ab1748f911c9ba65e527ecf4bfa1e27a3114`, including request schemas and create/update/read handlers. The provider still generates from the independently pinned `OnlineOrNot/api-schemas` revision in `schema.lock.json`. Examples are not defaults. No production API calls were used to verify this work.

## Coverage

All 17 registered resource types were reviewed. Existing generated scalar defaults remain generated. Missing array defaults and lifecycle-specific changes live in handwritten `Schema` methods, so regeneration does not overwrite them.

| Resources | Explicit request defaults and disposition |
| --- | --- |
| `check`, `uptime_check`, `browser_check` | `alert_priority = "LOW"`, `confirmation_period_seconds = 60`, `recovery_period_seconds = 180`, `reminder_alert_interval_minutes = 1440`; HTTP `method = "GET"`, `follow_redirects = true`, `verify_ssl = false`, `type = "UPTIME_CHECK"`. Typed browser forces `BROWSER_CHECK`. Existing generated defaults retained. |
| All five check resources | `timeout = 10000`. HTTP/browser retain the existing conditional plan override: scripted browser checks require omitted timeout and plan null because the API stores null. DNS/TCP retain static timeout defaults. |
| `dns_check` | Common monitor defaults above; `dns_protocol = "UDP"` retained. |
| `tcp_check` | Common monitor defaults above; `tcp_should_fail = false` retained; `tcp_ip_family` corrected from `Any` to the explicit API default `IPv4`. `Any` remains a valid explicit value. |
| `heartbeat` | `reminder_alert_interval_minutes = 1440`, `alert_priority = "LOW"` retained. Read now refreshes both fields. |
| `maintenance_window` | `checks = []`, `heartbeats = []` added as static list defaults. Reads normalize absent arrays to empty; JSON sends empty arrays rather than omitting them. |
| `status_page_component` | `status = "OPERATIONAL"`, `display_uptime = true`, `display_metrics = true` retained. Boolean pointers already preserve explicit false. Relationship and operational fields have no static defaults. |
| `status_page_incident` | `notify_subscribers = true` retained. Create-only status, description, components and notification settings retain replacement semantics because incident PATCH only accepts title/impact. No invented impact default. |
| `status_page_scheduled_maintenance` | Nested `notifications.an_hour_before = false`, `at_start = true`, `at_end = true` retained. There is no explicit parent object default. Do not synthesize an object from child defaults. The runtime falls back to these values when the whole object is omitted, but the existing parent Optional+Computed behavior is retained. Notifications, description and component membership are replacement-only because PATCH does not accept them. |
| `status_page` | No explicit defaults. Description, custom domain, allowed IPs and password are now optional-only with lifecycle handling described below. Search visibility remains Optional+Computed, resolved from API responses; its `false` example is not a schema default. ID is read-only and retained during update planning. |
| `webhook`, `status_page_component_group` | No explicit defaults. Existing Optional+Computed fields retained rather than converting fields whose omission, clearing and import behavior is not comprehensively handled by their CRUD implementations. |
| Three status-page ordering resources | No explicit defaults. Required ordered IDs are managed explicitly; resource identity is derived. Empty configured ordering remains meaningful. |
| `token` | No new static API defaults. Existing provider-only `never_expires = false` selects lifecycle behavior. Expiry defaults are relative to creation time, not static Terraform values. Secret and actual expiry remain read-only, with the existing create/import asymmetry. |

Source request definitions are under [`packages/shared-types/src/schemas`](https://github.com/OnlineOrNot/onlineornot-next/tree/d220ab1748f911c9ba65e527ecf4bfa1e27a3114/packages/shared-types/src/schemas): `checks.ts`, `heartbeats.ts`, `maintenance-windows.ts`, `statuspages.ts`, `statuspages-components.ts`, `statuspages-incidents.ts`, and `statuspages-scheduled-maintenance.ts`.

## Status-page lifecycle

The source create handler inserts name, subdomain, custom domain and password, but ignores description, allowed IPs and search visibility. The provider therefore follows creation with POST to the update endpoint when those settings were configured. It saves the new ID before that call. A failed settings update reports an error and may taint the resource; saving the ID avoids losing track of the created page, not Terraform's taint behavior.

The update client previously used PATCH despite the contract requiring POST. The new request type distinguishes omission from explicit false, empty strings and empty lists.

- Removing description sends `""`; removing allowed IPs sends `[]`. API omission would preserve their old values.
- Removing custom domain omits the property, which this API interprets as clearing it. Empty domain strings are not valid URL inputs. Reads preserve a configured URL when the API's normalized hostname matches, but report actual domain changes.
- Removing a previously managed password sends `""`. Omitted imported/unmanaged passwords remain omitted because the API does not return the secret. Password state is marked sensitive.
- Reads normalize API null descriptions/IP lists to Terraform null when omitted, and preserve explicitly configured empty values. Nonempty remote changes are still refreshed. Import hydrates readable fields, but cannot recover passwords.

Implementation evidence: [`workers/auth/src/lib/statuspages`](https://github.com/OnlineOrNot/onlineornot-next/tree/d220ab1748f911c9ba65e527ecf4bfa1e27a3114/workers/auth/src/lib/statuspages), particularly `create.ts`, `update.ts`, and `read.ts`.

## Zero values and runtime differences

Confirmation, recovery and reminder periods no longer use `omitempty` in check, DNS/TCP and heartbeat request structs. These fields have planned defaults, so a zero reaching the client is an explicit value, not an instruction to select the API default. Reads retain zero periods and the negative reminder-disable sentinel instead of turning them into null. Existing boolean pointers retain false; optional strings and collections without defaults elsewhere were not blanket-converted.

The check runtime has fallback values that differ from schema metadata: alert priority falls back to HIGH, and SSL verification to true. The provider explicitly sends the documented LOW and false defaults, so it does not depend on those fallback branches. HTTP update schemas remove most create defaults; Terraform still sends the planned full resource settings. Script timeout remains the deliberate exception.

No default was inferred from `test_interval = 60`, test-region examples, status-page search-visibility examples, server status, last-queued timestamps or identifiers. Existing Optional+Computed fields without defaults outside the status-page change remain conservative until their request omission, null/empty and read/import contracts have dedicated lifecycle coverage. This audit does not claim to eliminate every known-after-apply field or repair every existing CRUD inconsistency.

## Exclusions

Unsupported resources have explicit defaults but no registered Terraform resource to configure: integration Pushover failure/reminder priorities 1 and recovery priority 0; invitation role STANDARD; subscriber component subscription mode ALL_COMPONENTS and component IDs []; incident-update creation notify_subscribers true. Source calls the subscriber field `subscription_type`, while the pinned exported contract calls it `component_subscription_mode`. That mismatch does not change a supported resource.

Response-envelope success defaults and pagination query/result defaults 1/20 are not resource configuration. Data sources remain read-only. Runtime-relative token expiry and create/update-incompatible fields are not candidates for unconditional static defaults.

## Regression checks

Loopback Terraform tests check pre-apply known defaults for HTTP/uptime/browser/DNS/TCP checks, heartbeat, maintenance windows, components, incidents and nested scheduled-maintenance notifications. Status-page tests cover omitted values without unknown planning noise, configured values, clearing, password removal, domain normalization, refresh and import. Check lifecycle tests cover explicit zero periods, disabled reminders and false booleans. Raw JSON tests separately verify that zero periods and empty default arrays are actually sent, not merely retained in state. Existing tests still cover post-create unknown resolution and scripted browser timeout behavior.

All tests use mock servers or plan-only configurations. Live acceptance tests, deployments and production applies are outside this audit.
