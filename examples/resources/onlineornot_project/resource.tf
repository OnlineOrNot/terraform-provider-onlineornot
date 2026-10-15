resource "onlineornot_project" "staging" {
  name = "staging"
}

resource "onlineornot_uptime_check" "staging" {
  name       = "Staging API"
  url        = "https://staging.example.com/health"
  project_id = onlineornot_project.staging.id
}

resource "onlineornot_heartbeat" "staging" {
  name          = "Staging backup"
  grace_period  = 60
  report_period = 3600
  project_id    = onlineornot_project.staging.id
}

data "onlineornot_checks" "staging" {
  project_id = onlineornot_project.staging.id
}
