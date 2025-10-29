# Terraform Provider for Atlassian Compass

[![Go Report Card](https://goreportcard.com/badge/github.com/OSapozhnikov/terraform-provider-atlassian-compass)](https://goreportcard.com/report/github.com/OSapozhnikov/terraform-provider-atlassian-compass)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Terraform provider for managing Atlassian Compass components and links using the GraphQL API.

## Table of Contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Authentication](#authentication)
- [Usage](#usage)
- [Resources](#resources)
- [Provider Configuration](#provider-configuration)
- [Examples](#examples)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 0.13
- [Go](https://golang.org/doc/install) >= 1.21 (when building from source)
- Atlassian account with Compass access
- API token from [Atlassian API Tokens](https://id.atlassian.com/manage/api-tokens)

## Installation

### Using Terraform Registry (Recommended)

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

### Manual Installation

1. Download the latest release from the [releases page](https://github.com/OSapozhnikov/terraform-provider-atlassian-compass/releases)
2. Extract the binary to your Terraform plugins directory:
   - **Windows:** `%APPDATA%\terraform.d\plugins\registry.terraform.io\temabit\compass\1.0.0\windows_amd64\`
   - **Linux:** `~/.terraform.d/plugins/registry.terraform.io/OSapozhnikov/atlassian-compass/1.0.0/linux_amd64/`
   - **macOS:** `~/.terraform.d/plugins/registry.terraform.io/OSapozhnikov/atlassian-compass/1.0.0/darwin_amd64/`

## Authentication

The provider uses Basic Authentication with your Atlassian email and API token.

### Getting an API Token

1. Go to [Atlassian API Tokens](https://id.atlassian.com/manage/api-tokens)
2. Click "Create API token"
3. Give it a descriptive name (e.g., "Terraform Compass Provider")
4. Click "Create"
5. Copy the generated token (it will only be shown once!)

### Provider Configuration

```hcl
provider "compass" {
  email     = "your-email@example.com"
  api_token = "your-api-token"
  tenant    = "your-tenant"  # Optional: e.g., "your-tenant" for your-tenant.atlassian.net
  base_url  = "https://api.atlassian.com"  # Optional, defaults to https://api.atlassian.com
}
```

**Environment Variables:**

You can also use environment variables instead of hardcoding credentials:

```bash
export COMPASS_EMAIL="your-email@example.com"
export COMPASS_API_TOKEN="your-api-token"
export COMPASS_TENANT="your-tenant"
export COMPASS_BASE_URL="https://api.atlassian.com"  # Optional
```

### Cloud ID Detection

The provider can automatically detect your Cloud ID from your tenant name. If you provide the `tenant` parameter (e.g., "your-tenant"), the provider will automatically query the GraphQL API to get the Cloud ID for `your-tenant.atlassian.net`. You can also manually specify `cloud_id` in resources if needed.

## Usage

### Basic Example

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
  owner_id    = "your-account-id"  # Optional
}

resource "compass_component_link" "repository" {
  component_id = compass_component.example.id
  name         = "Repository"
  type         = "REPOSITORY"
  url          = "https://github.com/example/repo"
}
```

## Resources

### `compass_component`

Manages a Compass component.

See [component documentation](docs/resources/component.md) for full details.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | `string` | Yes | Name of the Compass component |
| `type` | `string` | Yes | Type of component. Valid values: `SERVICE`, `LIBRARY`, `APPLICATION`, `INFRASTRUCTURE`, `DATABASE`, `DOCUMENTATION` |
| `description` | `string` | No | Description of the component |
| `owner_id` | `string` | No | Owner ID (Atlassian account ID) of the component |
| `cloud_id` | `string` | No | Cloud ID. If not provided, will be auto-detected from tenant |

**Attributes:**

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | The unique identifier (ID) of the component |
| `cloud_id` | `string` | Cloud ID (computed if not provided) |

### `compass_component_link`

Manages a link attached to a Compass component.

See [component_link documentation](docs/resources/component_link.md) for full details.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `component_id` | `string` | Yes | ID of the Compass component to attach the link to |
| `name` | `string` | Yes | Name of the link |
| `type` | `string` | Yes | Type of the link. Valid values: `DOCUMENT`, `CHAT_CHANNEL`, `REPOSITORY`, `PROJECT`, `DASHBOARD`, `ON_CALL`, `OTHER_LINK` |
| `url` | `string` | Yes | URL of the link |
| `object_id` | `string` | No | Unique ID of the object the link points to (generally configured by integrations) |
| `cloud_id` | `string` | No | Cloud ID. If not provided, will be auto-detected from tenant |

**Attributes:**

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | The unique identifier (ID) of the link |
| `cloud_id` | `string` | Cloud ID (computed if not provided) |

## Provider Configuration

### Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `email` | `string` | Yes | Email address of your Atlassian account |
| `api_token` | `string` | Yes | API token for Atlassian Compass. Get it from [Atlassian API Tokens](https://id.atlassian.com/manage/api-tokens) |
| `tenant` | `string` | No | Tenant name for automatic cloud_id detection (e.g., 'temabit' for temabit.atlassian.net) |
| `base_url` | `string` | No | Base URL for Atlassian Compass GraphQL API. Defaults to `https://api.atlassian.com` |

## Examples

### Create a Component with Multiple Links

```hcl
resource "compass_component" "web_service" {
  name        = "Web Service"
  description = "Main web application service"
  type        = "SERVICE"
}

resource "compass_component_link" "gitlab_repo" {
  component_id = compass_component.web_service.id
  name         = "GitLab Repository"
  type         = "REPOSITORY"
  url          = "https://gitlab.com/example/web-service"
}

resource "compass_component_link" "documentation" {
  component_id = compass_component.web_service.id
  name         = "Documentation"
  type         = "DOCUMENT"
  url          = "https://docs.example.com/web-service"
}

resource "compass_component_link" "slack_channel" {
  component_id = compass_component.web_service.id
  name         = "Team Channel"
  type         = "CHAT_CHANNEL"
  url          = "https://slack.com/channels/web-service"
}
```

### Import Existing Resources

```bash
# Import a component
terraform import compass_component.example ari:cloud:compass:...:component/...

# Import a component link (format: component_id:link_id)
terraform import compass_component_link.repository ari:cloud:compass:...:component/...:1d1bd8b7-2834-438b-b9e3-b63156c57bf3
```

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/OSapozhnikov/terraform-provider-atlassian-compass.git
cd terraform-provider-atlassian-compass

# Download dependencies
go mod download

# Build the provider
go build -o terraform-provider-compass

# For Windows
go build -o terraform-provider-compass.exe
```

### Running in Debug Mode

```bash
go run main.go -debug
```

In another terminal, run Terraform:

```bash
terraform init
terraform plan -out="./terraform.tfplan"
terraform apply "./terraform.tfplan"
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with verbose output
go test -v ./...
```

### Code Formatting

```bash
go fmt ./...
```

## API Information

This provider uses the **Atlassian Compass GraphQL API** (`https://api.atlassian.com/graphql`). It does not use REST endpoints.

**Important Notes:**
- All operations use GraphQL mutations and queries
- Authentication is done via Basic Auth (email:token encoded in Base64)
- The provider includes the `X-ExperimentalApi: compass-beta` header for beta features
- Cloud ID can be auto-detected from tenant name using the `tenantContexts` query

**Documentation:**
- [Atlassian Compass GraphQL API](https://developer.atlassian.com/cloud/compass/graphql/)
- [Create Component Mutation](https://developer.atlassian.com/cloud/compass/graphql/#mutations_createComponent)
- [Update Component Mutation](https://developer.atlassian.com/cloud/compass/graphql/#mutations_updateComponent)
- [Create Component Link Mutation](https://developer.atlassian.com/cloud/compass/graphql/#mutations_createComponentLink)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues:** [GitHub Issues](https://github.com/OSapozhnikov/terraform-provider-atlassian-compass/issues)
- **Documentation:** [docs/](docs/)

## Author

Developed by **Temabit**

**Repository:** [https://github.com/OSapozhnikov/terraform-provider-atlassian-compass](https://github.com/OSapozhnikov/terraform-provider-atlassian-compass)

