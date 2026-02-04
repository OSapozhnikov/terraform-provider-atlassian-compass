terraform {
  required_version = ">= 1.14"
  required_providers {
    # compass = {
    #   source  = "OSapozhnikov/atlassian-compass"
    #   version = "1.0.8"
    # }
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
  type        = "CAPABILITY"
  slug        = "terraform-test-capability"
}

resource "compass_component_labels" "example" {
  component_id = compass_component.example.id
  labels  = ["terraform-test-label", "terraform-test-label-3"]
}

resource "compass_component_link" "repository" {
  component_id = compass_component.example.id
  name         = "Terraform created test component link"
  type         = "REPOSITORY"
  url          = var.compass_component_repository_url
}

resource "compass_component_relationship" "depends_on" {
  start_node_id     = compass_component.example.id
  end_node_id       = var.compass_component_target_depends
  relationship_type = "DEPENDS_ON"
}

resource "compass_component_relationship" "child_of" {
  start_node_id     = compass_component.example.id
  end_node_id       = var.compass_component_target_child
  relationship_type = "CHILD_OF"
}