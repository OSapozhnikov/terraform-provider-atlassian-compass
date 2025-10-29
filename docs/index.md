# Terraform Provider: Atlassian Compass

This provider lets you manage Atlassian Compass resources using the Compass GraphQL API. It follows a structure similar to official Terraform provider docs (e.g., the GitLab provider style) for clarity and consistency.

## Requirements

- Terraform >= 0.13
- Atlassian account with Compass access
- Atlassian API token

## Installation

Add the provider to your Terraform configuration:

```hcl
terraform {
  required_providers {
    compass = {
      source  = "OSapozhnikov/atlassian-compass"
      version = "~> 1.0"
    }
  }
}
```

Then initialize:

```bash
terraform init
```

## Authentication

The provider uses Basic Authentication with your Atlassian email and API token.

Provider configuration example:

```hcl
provider "compass" {
  email     = var.compass_email
  api_token = var.compass_api_token
  tenant    = var.compass_tenant          # Optional (e.g., 'your-tenant' for your-tenant.atlassian.net)
  base_url  = "https://api.atlassian.com" # Optional, default is https://api.atlassian.com
}
```

Environment variables:

```bash
export COMPASS_EMAIL="your-email@example.com"
export COMPASS_API_TOKEN="your-api-token"
export COMPASS_TENANT="your-tenant"
export COMPASS_BASE_URL="https://api.atlassian.com"
```

Cloud ID detection: If `tenant` is provided, the provider can auto-detect Cloud ID via GraphQL. You can also specify `cloud_id` directly on resources when needed.

## Usage

Minimal example:

```hcl
terraform {
  required_providers {
    compass = {
      source  = "OSapozhnikov/atlassian-compass"
      version = "~> 1.0"
    }
  }
}

provider "compass" {
  email     = var.compass_email
  api_token = var.compass_api_token
  tenant    = var.compass_tenant
}

resource "compass_component" "example" {
  name        = "My Service"
  description = "A sample Compass component"
  type        = "SERVICE"
}

resource "compass_component_link" "repository" {
  component_id = compass_component.example.id
  name         = "Repository"
  type         = "REPOSITORY"
  url          = "https://gitlab.com/example/repo"
}
```

## Resources

- `compass_component` — Manages a Compass component
  - Full docs: [`docs/resources/component.md`](./resources/component.md)
- `compass_component_link` — Manages a link attached to a Compass component
  - Full docs: [`docs/resources/component_link.md`](./resources/component_link.md)

Quick references:

```hcl
resource "compass_component" "example" {
  name        = "My Service"
  description = "A sample Compass component"
  type        = "SERVICE"
}
```

```hcl
resource "compass_component_link" "repository" {
  component_id = compass_component.example.id
  name         = "Repository"
  type         = "REPOSITORY"
  url          = "https://gitlab.com/example/repo"
}
```

## Import

Import existing resources into Terraform state.

```bash
# Import a component (use the component ARI)
terraform import compass_component.example ari:cloud:compass:...:component/...

# Import a component link (format: component_id:link_id)
terraform import compass_component_link.repository ari:cloud:compass:...:component/...:1d1bd8b7-2834-438b-b9e3-b63156c57bf3
```

## GraphQL API

This provider uses the Atlassian Compass GraphQL API.

- Endpoint: `https://api.atlassian.com/graphql`
- Documentation:
  - Atlassian Compass GraphQL API: https://developer.atlassian.com/cloud/compass/graphql/
  - Create Component Mutation: https://developer.atlassian.com/cloud/compass/graphql/#mutations_createComponent
  - Update Component Mutation: https://developer.atlassian.com/cloud/compass/graphql/#mutations_updateComponent
  - Create Component Link Mutation: https://developer.atlassian.com/cloud/compass/graphql/#mutations_createComponentLink

Notes:
- All operations are GraphQL queries/mutations
- Authentication via Basic Auth (email:token in Base64)
- The provider may set `X-ExperimentalApi: compass-beta` for beta features
- Cloud ID can be auto-detected from tenant via GraphQL queries
