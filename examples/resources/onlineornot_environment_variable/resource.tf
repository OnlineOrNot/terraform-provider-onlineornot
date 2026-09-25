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

# Use the computed reference in a check with environment variables. The secret
# itself remains write-only; headers contain only its non-sensitive template.
resource "onlineornot_check" "api" {
  name = "API Health Check"
  url  = "https://api.example.com/health"

  headers = {
    env-test      = onlineornot_environment_variable.api_token.reference
    Authorization = "Bearer ${onlineornot_environment_variable.api_token.reference}"
  }
}
