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
  name        = "Terraform Test Component"
  description = "This is a Terraform created test component"
  type        = "SERVICE"
}

resource "compass_component_link" "repository" {
  component_id = compass_component.example.id
  name         = "Terraform created test component link"
  type         = "REPOSITORY"
  url          = var.compass_component_repository_url
}
