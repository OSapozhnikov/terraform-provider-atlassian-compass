# compass_component_types

Data source for querying available component types in Atlassian Compass for a given site (cloud_id).

This allows you to:
- inspect all built-in and custom component types available in your Compass catalog
- look up a specific type by `id` (enum-style key or ARI)
- look up a specific type by human‑readable `name`

## Example Usage

### List all component types

```hcl
data "compass_component_types" "all" {}

output "component_types" {
  value = data.compass_component_types.all.types
}
```

### Find a built‑in type by id

```hcl
data "compass_component_types" "service" {
  id = "SERVICE"
}

output "service_type" {
  value = data.compass_component_types.service.types[0]
}
```

### Find a custom type by id (ARI)

```hcl
data "compass_component_types" "domain" {
  id = "ari:cloud:compass:...:component-type/..."
}

output "domain_type" {
  value = data.compass_component_types.domain.types[0]
}
```

### Find a type by name

```hcl
data "compass_component_types" "domain_by_name" {
  name = "Domain"
}

output "domain_type_by_name" {
  value = data.compass_component_types.domain_by_name.types[0]
}
```

## Argument Reference

The following arguments are supported:

* `cloud_id` - (Optional, Computed) Cloud ID of the Atlassian site. If not provided, it will be automatically detected from the `tenant` configured in the provider.
* `id` - (Optional) Filter to return only the component type with this identifier.
  * For built‑in types this is the enum‑style key (for example `SERVICE`, `LIBRARY`, `APPLICATION`, `CAPABILITY`, `CLOUD_RESOURCE`, `DATA_PIPELINE`, `MACHINE_LEARNING_MODEL`, `UI_ELEMENT`, `WEBSITE`, `OTHER`).
  * For custom types this is the full component type ARI.
* `name` - (Optional) Filter to return only component types with this human‑readable name (for example `Service`, `Domain`, `Product`).

**Notes:**

* At most **one** of `id` or `name` may be specified. If both are provided, the data source will return an error.
* If neither `id` nor `name` is specified, all available component types for the given `cloud_id` are returned.

If a filter is provided and no matching component type is found, the data source will return an error indicating that no component type with the given `id` or `name` exists for the specified `cloud_id`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `cloud_id` - The Cloud ID that was used when querying the Compass GraphQL API.
* `types` - List of component type objects for the given `cloud_id`. Each object has the following attributes:
  * `id` - Identifier of the component type.
    * For built‑in types this is the enum‑style key (for example `SERVICE`, `LIBRARY`, `APPLICATION`, `CAPABILITY`, `CLOUD_RESOURCE`, `DATA_PIPELINE`, `MACHINE_LEARNING_MODEL`, `UI_ELEMENT`, `WEBSITE`, `OTHER`).
    * For custom types this is the full component type ARI.
  * `name` - Human‑readable name of the component type (for example `Service`, `Domain`, `Product`).

