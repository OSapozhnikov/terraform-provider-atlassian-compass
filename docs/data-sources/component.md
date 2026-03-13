# compass_component

Data source for retrieving a single Compass component by ID (ARI or internal ID) or by slug.

Lookup uses the Compass GraphQL API:
- **By ID:** Query `component(id)` — use the `id` argument.
- **By slug:** Query `componentByReference(reference: { slug: { slug, cloudId } })` — use the `slug` argument; `cloud_id` is required unless `tenant` is set in the provider.

Exactly **one** of `id` or `slug` must be set.

## Example Usage

### Look up by ID (ARI)

```hcl
data "compass_component" "by_id" {
  id = "ari:cloud:compass:a1250265-f505-432c-90ff-5d28665aa42c:component/c25b7bb8-a5d0-4b6e-b577-79f4d9bc530e/e8d4a0fe-19d0-41a0-9130-a6967a0c804d"
}

output "component_name" {
  value = data.compass_component.by_id.name
}

output "component_url" {
  value = data.compass_component.by_id.url
}
```

### Look up by slug (with tenant in provider)

```hcl
provider "compass" {
  email     = var.compass_email
  api_token = var.compass_api_token
  tenant    = "temabit"  # cloud_id will be auto-detected for slug lookup
}

data "compass_component" "by_slug" {
  slug = "temabit-product-example"
}

output "component_id" {
  value = data.compass_component.by_slug.id
}
```

### Look up by slug (with explicit cloud_id)

```hcl
data "compass_component" "by_slug" {
  slug     = "temabit-product-example"
  cloud_id = "a1250265-f505-432c-90ff-5d28665aa42c"
}

output "component_description" {
  value = data.compass_component.by_slug.description
}
```

## Argument Reference

The following arguments are supported:

* `id` - (Optional, Computed) ARI or internal ID of the Compass component. Use for lookup via Query `component(id)`. After read, holds the component's ARI. Exactly one of `id` or `slug` must be set.
* `slug` - (Optional, Computed) Slug of the Compass component. Use for lookup via Query `componentByReference`. After read, holds the component's slug. Exactly one of `id` or `slug` must be set.
* `cloud_id` - (Optional, Computed) Cloud ID of the Atlassian site. Required for slug lookup when `tenant` is not configured in the provider; otherwise auto-detected from `tenant`.

**Notes:**

* You must set **exactly one** of `id` or `slug`. Setting both or neither will return an error.
* For slug lookup, either set `cloud_id` or configure `tenant` in the provider so the provider can resolve `cloud_id` automatically.

## Attributes Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The component's ARI (Atlassian Resource Identifier).
* `name` - Name of the component.
* `slug` - Slug of the component.
* `description` - Description of the component.
* `url` - URL to the component in Compass.
* `type_id` - Type ID of the component (e.g. `SERVICE`, or custom type ARI).
* `owner_id` - Owner team or user ID of the component.
