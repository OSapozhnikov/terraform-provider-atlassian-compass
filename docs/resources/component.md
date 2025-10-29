# compass_component

Manages a Compass component in Atlassian Compass.

## Example Usage

### Basic Component

```hcl
resource "compass_component" "example" {
  name        = "My Service"
  description = "A sample Compass component"
  type        = "SERVICE"
}
```

### Component with Owner

```hcl
resource "compass_component" "example" {
  name        = "My Service"
  description = "A sample Compass component"
  type        = "SERVICE"
  owner_id    = "557058:e6b5f8e8-1234-5678-9abc-def012345678"
}
```

### Component with Explicit Cloud ID

```hcl
resource "compass_component" "example" {
  cloud_id    = "a1250265-f505-432c-90ff-5d28665aa42c"
  name        = "My Service"
  description = "A sample Compass component"
  type        = "SERVICE"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the Compass component.
* `type` - (Required) Type of the Compass component. Valid values are:
  * `SERVICE` - A service component
  * `LIBRARY` - A library component
  * `APPLICATION` - An application component
  * `INFRASTRUCTURE` - An infrastructure component
  * `DATABASE` - A database component
  * `DOCUMENTATION` - A documentation component
* `description` - (Optional) Description of the Compass component.
* `owner_id` - (Optional) Owner ID (Atlassian account ID) of the Compass component. This should be the account ID of the user or team that owns the component.
* `cloud_id` - (Optional, Computed) Cloud ID of the Atlassian site (e.g., `jira-12345678-1234-1234-1234-123456789012`). If not provided, will be automatically detected from the `tenant` configured in the provider.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier (ID) of the component in Atlassian Resource Identifier (ARI) format. Example: `ari:cloud:compass:a1250265-f505-432c-90ff-5d28665aa42c:component/c25b7bb8-a5d0-4b6e-b577-79f4d9bc530e/0bc38bf0-4c91-4d91-a3d1-9cf38fa6150c`
* `cloud_id` - Cloud ID (computed if not provided explicitly)

## Import

Components can be imported using their ARI identifier:

```bash
terraform import compass_component.example ari:cloud:compass:a1250265-f505-432c-90ff-5d28665aa42c:component/c25b7bb8-a5d0-4b6e-b577-79f4d9bc530e/0bc38bf0-4c91-4d91-a3d1-9cf38fa6150c
```

Alternatively, you can import using just the component ID part:

```bash
terraform import compass_component.example c25b7bb8-a5d0-4b6e-b577-79f4d9bc530e/0bc38bf0-4c91-4d91-a3d1-9cf38fa6150c
```

## Update Behavior

The resource supports updating the following fields:
* `name` - Can be updated
* `description` - Can be updated
* `owner_id` - Can be updated

**Fields that cannot be updated:**
* `type` - Component type cannot be changed after creation. You must delete and recreate the component with the new type.
* `cloud_id` - Cloud ID cannot be changed after creation.

## Notes

* The component ID returned by the API is in ARI (Atlassian Resource Identifier) format and contains the component's unique identifier.
* If `cloud_id` is not provided, the provider will automatically detect it from the `tenant` parameter in the provider configuration using the GraphQL `tenantContexts` query.
* The `owner_id` should be the Atlassian account ID of the user or team that owns the component. This can be found in your Atlassian profile or via the GraphQL API.

