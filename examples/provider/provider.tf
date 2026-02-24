terraform {
  required_providers {
    onlineornot = {
      source = "onlineornot/onlineornot"
    }
  }
}

# Configure the provider using the ONLINEORNOT_API_KEY environment variable
provider "onlineornot" {}

# Or configure with explicit API key (not recommended for production)
# provider "onlineornot" {
#   api_key = "your-api-key"
# }
