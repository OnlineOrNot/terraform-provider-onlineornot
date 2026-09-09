terraform {
  required_providers {
    onlineornot = {
      source  = "onlineornot/onlineornot"
      version = "~> 0.1.24"
    }
  }
}

provider "onlineornot" {
  # Set ONLINEORNOT_API_KEY to your API token.
}
