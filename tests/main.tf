terraform {
  required_providers {
    compass = {
      source  = "OSapozhnikov/atlassian-compass"
      version = "1.0.7"
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
  name        = "Terraform Test Library"
  description = "This is a Terraform created test LIBRARY component"
  type        = "LIBRARY"
}

resource "compass_component_link" "repository" {
  component_id = compass_component.example.id
  name         = "Terraform created test component link"
  type         = "REPOSITORY"
  url          = var.compass_component_repository_url
}
