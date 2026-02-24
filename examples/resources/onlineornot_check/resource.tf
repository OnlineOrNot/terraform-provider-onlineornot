# Basic uptime check
resource "onlineornot_check" "example" {
  name = "My Website"
  url  = "https://example.com"
}

# Uptime check with custom configuration
resource "onlineornot_check" "advanced" {
  name          = "API Health Check"
  url           = "https://api.example.com/health"
  method        = "GET"
  test_interval = 60
  timeout       = 5000

  # Alert configuration
  alert_priority                  = "HIGH"
  confirmation_period_seconds     = 30
  recovery_period_seconds         = 60
  reminder_alert_interval_minutes = 60

  # SSL and redirect settings
  verify_ssl       = true
  follow_redirects = true

  # Notify specific users
  user_alerts = [data.onlineornot_users.all.users[0].id]
}

# Browser check (requires Playwright)
resource "onlineornot_check" "browser" {
  name    = "Homepage Load Test"
  url     = "https://example.com"
  type    = "BROWSER_CHECK"
  version = "NODE20_PLAYWRIGHT"
}
