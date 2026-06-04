resource "onlineornot_uptime_check" "api" {
  name = "API health check"
  url  = "https://api.example.com/health"

  method        = "GET"
  test_interval = 60
  timeout       = 10000
  test_regions = [
    "aws:us-east-1",
    "aws:eu-central-1",
  ]

  assertions = [
    {
      type       = "TEXT_BODY"
      property   = ""
      comparison = "CONTAINS"
      expected   = "ok"
    }
  ]
}
