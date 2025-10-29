terraform {
  required_providers {
    compass = {
      source  = "temabit/compass"
      version = "1.0.0"
    }
  }
}

provider "compass" {
  email     = var.compass_email
  api_token = var.compass_api_token
  tenant    = var.compass_tenant
}

### Create component

resource "compass_component" "example" {
  name        = "My Service Component"
  description = "This is a sample service component"
  type        = "SERVICE"
}