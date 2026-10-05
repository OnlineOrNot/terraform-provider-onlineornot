resource "onlineornot_status_page" "example" {
  name      = "Example Status"
  subdomain = "example-status"

  # Read local files and upload their contents, not their filesystem paths.
  logo      = "data:image/svg+xml;base64,${filebase64("${path.module}/logo.svg")}"
  dark_logo = "data:image/svg+xml;base64,${filebase64("${path.module}/logo-dark.svg")}"
  favicon   = "data:image/svg+xml;base64,${filebase64("${path.module}/favicon.svg")}"
}
