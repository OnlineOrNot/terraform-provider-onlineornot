terraform {
  required_providers {
    onlineornot = {
      source  = "onlineornot/onlineornot"
      version = "~> 0.1.24"
    }
  }
}

provider "onlineornot" {
  # Supply the token through ONLINEORNOT_API_KEY.
}
