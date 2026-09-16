---
page_title: "Secret request headers"
subcategory: "Checks"
description: |-
  Use ephemeral request headers without storing their values in Terraform plan or state.
---

# Secret request headers

`onlineornot_uptime_check`, `onlineornot_browser_check`, and the legacy
`onlineornot_check` support `write_only_headers`: an optional, sensitive,
write-only `map(string)`, mutually exclusive with ordinary `headers` (including
an empty ordinary map). Terraform **1.11 or later** is required to use it.

```terraform
terraform {
  required_version = ">= 1.11.0"
  required_providers {
    onlineornot = {
      source = "onlineornot/onlineornot"
    }
  }
}

variable "api_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "onlineornot_uptime_check" "api" {
  name = "Authenticated health check"
  url  = "https://api.example.com/health"

  write_only_headers = {
    Authorization = "Bearer ${var.api_token}"
    Accept        = "application/json"
  }
  write_only_headers_version = 1
}
```

Supply `TF_VAR_api_token` through your execution environment's secret manager,
including during apply of a saved plan. Do not hardcode secrets or use ordinary
(non-ephemeral) variables: those can persist elsewhere in Terraform artifacts.
`Sensitive` alone only redacts display; it does not prevent storage.

## Rotation and clearing

- Configure both write-only arguments together. The version is a positive,
  **non-secret integer stored in state**. Increment it to rotate headers.
- Headers are sent on creation and when the version changes. Changing only the
  secret does not produce a diff or resend it, even during an unrelated update.
  Terraform cannot detect remote drift in these values.
- Send `write_only_headers = {}` with a new version to explicitly clear headers
  while retaining write-only mode. Null map values are not allowed.
- Removing both arguments clears remote headers on the next apply. Replacing
  them with ordinary `headers` replaces the remote map and stores those ordinary
  values in state. Switching from ordinary to write-only headers sends the full
  new map and removes ordinary headers from the current state.
- The maps cannot be mixed. Include non-secret headers in `write_only_headers`
  too when a request needs both kinds.

## Import and migration

The API does not label secret headers. **Imports therefore never populate
ordinary `headers`**, even for monitors previously using ordinary headers.
Import preserves remote headers; unrelated updates leave them alone. After
import, configure both write-only arguments and apply to establish management
and send your supplied values. To manage non-secret headers instead, explicitly
configure the complete ordinary `headers` map and apply. An empty ordinary map
clears previously unmanaged headers.

Refresh preserves ordinary-header drift detection for resources already
managing an ordinary map. Do not put secrets into a monitor managed through
ordinary `headers`, including via an external API update: those values will
still be read into state. Avoid managing the same monitor from multiple states
or older provider versions.

Migration does **not** scrub historical state snapshots, saved plans, logs, or
backups containing old ordinary headers. Rotate exposed credentials and follow
your backend's retention/removal procedures. Removing provider-managed state
and reimporting also loses the rotation version; supply it and apply again.

## Security boundary

OnlineOrNot still receives and stores these headers so it can run the monitor;
this feature protects Terraform plan/state storage, not the backend API. No raw
header values are stored in provider private state. Check resource API error
details are withheld because responses may echo secrets.

The `onlineornot_checks` data source exposes only ID, name, URL, check type,
status, and method—not headers. Other attributes (including URL, body, basic-auth
passwords, scripts, and assertions) are **not** made write-only by this feature.
Do not copy secret header values into those fields or outputs. Restrict debug
logging and access to the API and Terraform execution environment.
