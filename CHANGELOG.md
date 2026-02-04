# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-02-04

### Added

- **Data source `compass_component_types`** — query available Compass component types for a site (`cloud_id`):
  - Returns a list of types with `id` and `name` (built-in and custom).
  - Optional filters: `id` (type id or ARI) or `name` (human-readable); at most one filter. Clear error when no type matches.
  - Optional `cloud_id`; auto-detected from provider `tenant` when omitted.
  - Documentation under `docs/data-sources/component_types.md`.
- **`compass_component`: optional `slug`** — unique identifier for the component (UTF-8 string):
  - Supported in create (pass `null` when not set) and update (can be changed or cleared).
  - Returned in read and documented in resources/component and Update Behavior.

### Changed

- **`compass_component`: use `typeId` instead of deprecated `type` in GraphQL**:
  - Create mutation now sends `typeId` (ID) in the input; the deprecated `type` (CompassComponentType enum) is no longer used.
  - The Terraform attribute remains `type`: users still set a human-readable value (type **id** such as `SERVICE`, `LIBRARY`, `APPLICATION`, or full ARI for custom types, or type **name** such as `Service`, `Domain`). The provider resolves it to `typeId` via the Compass `componentTypes` GraphQL query.
  - Resolution order: exact match by id, then by name, then case-insensitive name (error if multiple matches). On no match, error lists available types for the site.
  - Client: added `GetComponentTypes(ctx, cloudID)` with per-`cloud_id` in-memory cache for the process lifetime.
  - Supports all built-in and custom component types; no hard-coded type list.
  - Read/update: `type` in state is unchanged (user value preserved); `type` and `cloud_id` remain immutable after create.

### Documentation

- `docs/index.md`: added Data Sources section and link to `compass_component_types`.
- `docs/resources/component.md`: `type` description updated (id/name, componentTypes, link to data source); `slug` argument and Update Behavior (slug updatable).
- README: data source section for `compass_component_types`; component `type` and `slug` mentioned where relevant.

## [1.0.8] - 2025-10-29

### Added
- Unit tests for provider resources using an in-memory GraphQL mock server:
  - `compass_component` CRUD
  - `compass_component_link` CRUD (including import format `component_id:link_id`)

### CI
- GitHub Actions workflow updated to run `go test -v ./...` on pushes and PRs.
- Go toolchain updated to `1.24` in CI.

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
