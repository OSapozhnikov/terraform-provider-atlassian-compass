# Resources

This section provides detailed documentation for all resources supported by the Terraform Compass provider.

## Available Resources

* [`compass_component`](./resources/component.md) - Manages a Compass component in Atlassian Compass
* [`compass_component_link`](./resources/component_link.md) - Manages a link attached to a Compass component

## Quick Reference

### compass_component

Creates and manages Compass components.

**Quick Example:**

```hcl
resource "compass_component" "example" {
  name        = "My Service"
  description = "A sample Compass component"
  type        = "SERVICE"
}
```

**[Full Documentation →](./resources/component.md)**

### compass_component_link

Creates and manages links attached to Compass components (repositories, documentation, dashboards, etc.).

**Quick Example:**

```hcl
resource "compass_component_link" "repository" {
  component_id = compass_component.example.id
  name         = "Repository"
  type         = "REPOSITORY"
  url          = "https://gitlab.com/example/repo"
}
```

**[Full Documentation →](./resources/component_link.md)**

