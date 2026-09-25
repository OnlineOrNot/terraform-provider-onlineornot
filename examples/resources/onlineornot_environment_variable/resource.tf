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
