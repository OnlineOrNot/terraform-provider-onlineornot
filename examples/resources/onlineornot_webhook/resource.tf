# Create WEBHOOK_TOKEN in OnlineOrNot before using this reference.
# OnlineOrNot resolves {{NAME}} references when it sends the webhook.
resource "onlineornot_webhook" "example" {
  url       = "https://example.com/hooks/{{WEBHOOK_TOKEN}}"
  events    = ["uptime.down", "uptime.up"]
  check_ids = [onlineornot_uptime_check.example.id]
}

resource "onlineornot_uptime_check" "example" {
  name = "Example website"
  url  = "https://example.com"
}
