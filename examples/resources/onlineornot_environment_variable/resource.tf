terraform {
  required_version = ">= 1.11"
}

variable "api_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "onlineornot_environment_variable" "api_token" {
  name          = "API_TOKEN"
  type          = "secret"
  value         = var.api_token
  value_version = "rotation-1"
}

# .reference produces "{{API_TOKEN}}". .name produces only "API_TOKEN".
# The secret stays write-only. Headers contain the reference, not the secret.
resource "onlineornot_uptime_check" "api" {
  name = "API Health Check"
  url  = "https://api.example.com/health"

  headers = {
    env-test      = onlineornot_environment_variable.api_token.reference
    Authorization = "Bearer ${onlineornot_environment_variable.api_token.reference}"
  }
}
