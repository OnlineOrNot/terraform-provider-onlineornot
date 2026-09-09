# The provider key needs API_TOKENS EDIT and every grant delegated below.
# Omitting expiry uses the API default of 365 days from creation.
resource "onlineornot_token" "monitoring" {
  name = "Monitoring automation"
  grants = [
    { scope = "UPTIME_CHECKS", permission = "READ" },
    { scope = "STATUS_PAGES", permission = "READ" },
  ]
  expires_at = "2030-01-01T00:00:00Z"
  # For no expiration, omit expires_at and explicitly set never_expires = true.
}

output "monitoring_token" {
  value     = onlineornot_token.monitoring.token
  sensitive = true
}
