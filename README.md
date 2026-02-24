# Terraform Provider for OnlineOrNot

Manage your [OnlineOrNot](https://onlineornot.com) uptime monitoring infrastructure as code.

## Features

- **Uptime Checks** - Monitor websites and APIs with HTTP checks
- **Heartbeat Monitors** - Track cron jobs and background processes
- **Status Pages** - Create and manage public status pages
- **Incidents & Maintenance** - Declare incidents and scheduled maintenance windows
- **Webhooks** - Configure alert webhooks
- **Alert Routing** - Assign users to receive alerts via email, Slack, Discord, PagerDuty, and more

## Quick Start

```hcl
terraform {
  required_providers {
    onlineornot = {
      source = "onlineornot/onlineornot"
    }
  }
}

provider "onlineornot" {
  # Set via ONLINEORNOT_API_KEY environment variable, or:
  # api_key = "your-api-key"
}

# Look up a team member by email
data "onlineornot_user" "ops_lead" {
  email = "ops@example.com"
}

# Create an uptime check with alerts
resource "onlineornot_check" "api" {
  name        = "Production API"
  url         = "https://api.example.com/health"
  method      = "GET"
  user_alerts = [data.onlineornot_user.ops_lead.id]
}

# Create a status page
resource "onlineornot_status_page" "public" {
  name      = "Example Status"
  subdomain = "status-example"
}

# Add a component to the status page
resource "onlineornot_status_page_component" "api" {
  status_page_id = onlineornot_status_page.public.id
  name           = "API"
}
```

## Authentication

Get your API key from [OnlineOrNot API Tokens](https://onlineornot.com/app/api-tokens).

```bash
export ONLINEORNOT_API_KEY="your-api-key"
terraform plan
```

Or set it in the provider block:

```hcl
provider "onlineornot" {
  api_key = "your-api-key"  # Not recommended for version control
}
```

## Resources

| Resource | Description |
|----------|-------------|
| `onlineornot_check` | Uptime check (HTTP/HTTPS monitoring) |
| `onlineornot_heartbeat` | Heartbeat monitor for cron jobs |
| `onlineornot_status_page` | Public status page |
| `onlineornot_status_page_component` | Status page component |
| `onlineornot_status_page_component_group` | Group of components |
| `onlineornot_status_page_incident` | Status page incident |
| `onlineornot_status_page_scheduled_maintenance` | Scheduled maintenance window |
| `onlineornot_webhook` | Webhook for alerts |
| `onlineornot_maintenance_window` | Maintenance window (suppresses alerts) |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `onlineornot_user` | Look up a user by email or ID |
| `onlineornot_users` | List all users in your organisation |
| `onlineornot_checks` | List all uptime checks |
| `onlineornot_heartbeats` | List all heartbeat monitors |
| `onlineornot_status_pages` | List all status pages |
| `onlineornot_webhooks` | List all webhooks |
| `onlineornot_maintenance_windows` | List all maintenance windows |

## Examples

### Monitor multiple endpoints with shared alerts

```hcl
data "onlineornot_user" "oncall" {
  email = "oncall@example.com"
}

locals {
  endpoints = {
    api     = "https://api.example.com/health"
    web     = "https://example.com"
    admin   = "https://admin.example.com"
  }
}

resource "onlineornot_check" "endpoints" {
  for_each    = local.endpoints
  name        = "Production ${each.key}"
  url         = each.value
  user_alerts = [data.onlineornot_user.oncall.id]
}
```

### Status page with component groups

```hcl
resource "onlineornot_status_page" "main" {
  name      = "Acme Status"
  subdomain = "status-acme"
}

resource "onlineornot_status_page_component_group" "backend" {
  status_page_id = onlineornot_status_page.main.id
  name           = "Backend Services"
}

resource "onlineornot_status_page_component" "api" {
  status_page_id = onlineornot_status_page.main.id
  name           = "REST API"
}

resource "onlineornot_status_page_component" "database" {
  status_page_id = onlineornot_status_page.main.id
  name           = "Database"
}
```

## Development

### Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21

### Building

```bash
go build -o terraform-provider-onlineornot
```

### Local Testing

Create `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "onlineornot/onlineornot" = "/path/to/terraform-provider-onlineornot"
  }
  direct {}
}
```

Then run Terraform without `terraform init`:

```bash
export ONLINEORNOT_API_KEY="your-api-key"
terraform plan
```

## License

[MIT](LICENSE)
