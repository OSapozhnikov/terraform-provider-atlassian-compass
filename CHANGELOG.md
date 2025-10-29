# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.7] - 2025-10-29

### Added
- Initial release of Terraform Provider for Atlassian Compass
- Support for creating, reading, updating, and deleting Compass components
- Support for managing component links (repositories, documentation, dashboards, etc.)
- Automatic Cloud ID detection from tenant name
- Environment variable support for credentials
- Import functionality for existing resources
- Full GraphQL API integration with Compass
- Documentation: Updated `docs/index.md` to Terraform provider style (installation, authentication, usage, resources, import) and added GraphQL links.

### Resources
- `compass_component` - Manage Compass components
- `compass_component_link` - Manage links attached to components

### Features
- Basic authentication with email and API token
- Support for all component types: SERVICE, LIBRARY, APPLICATION, INFRASTRUCTURE, DATABASE, DOCUMENTATION
- Support for all link types: DOCUMENT, CHAT_CHANNEL, REPOSITORY, PROJECT, DASHBOARD, ON_CALL, OTHER_LINK
- Automatic tenant-to-cloud-id resolution
- Comprehensive error handling and validation
