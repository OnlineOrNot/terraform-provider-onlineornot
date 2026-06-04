resource "onlineornot_dns_check" "root_a" {
  name            = "example.com A record"
  dns_domain      = "example.com"
  dns_record_type = "A"
  dns_protocol    = "UDP"

  test_interval = 300
  test_regions = [
    "aws:us-east-1",
    "aws:eu-west-2",
  ]
}

resource "onlineornot_dns_check" "custom_resolver" {
  name            = "example.com via Cloudflare DNS"
  dns_domain      = "example.com"
  dns_record_type = "A"
  dns_resolver    = "1.1.1.1"
  dns_protocol    = "UDP"
}
