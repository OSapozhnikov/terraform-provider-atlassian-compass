# compass_component_relationship

Creates a directed relationship between two Compass components (e.g. "component A depends on component B" or "B is child of A"). Uses the Compass GraphQL mutations `createRelationship` and `deleteRelationship`.

## Example Usage

### DEPENDS_ON (dependency)

```hcl
resource "compass_component" "service_a" {
  name = "Service A"
  type = "SERVICE"
}

resource "compass_component" "service_b" {
  name = "Service B"
  type = "SERVICE"
}

resource "compass_component_relationship" "a_depends_on_b" {
  start_node_id     = compass_component.service_a.id
  end_node_id      = compass_component.service_b.id
  relationship_type = "DEPENDS_ON"
}
```

### CHILD_OF (parent-child)

```hcl
resource "compass_component_relationship" "child_of" {
  start_node_id     = compass_component.child.id
  end_node_id      = compass_component.parent.id
  relationship_type = "CHILD_OF"
}
```

### With explicit cloud_id

```hcl
resource "compass_component_relationship" "dep" {
  start_node_id     = compass_component.service_a.id
  end_node_id      = compass_component.service_b.id
  relationship_type = "DEPENDS_ON"
  cloud_id         = "a1250265-f505-432c-90ff-5d28665aa42c"
}
```

## Argument Reference

* `start_node_id` - (Required, ForceNew) ID of the component at the start of the relationship (e.g. the dependent, or the child).
* `end_node_id` - (Required, ForceNew) ID of the component at the end (e.g. the one we depend on, or the parent).
* `relationship_type` - (Required, ForceNew) Type of relationship. Valid values: `DEPENDS_ON`, `CHILD_OF`.
* `cloud_id` - (Optional, Computed, ForceNew) Cloud ID of the Atlassian site. If not provided, will be automatically detected from the `tenant` configured in the provider.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The relationship ID (from the API) or a composite of `start_node_id:end_node_id:relationship_type`.

## Import

Import by composite id `start_node_id:end_node_id:relationship_type`:

```bash
terraform import compass_component_relationship.example "ari:cloud:compass:...:component/...:ari:cloud:compass:...:component/...:DEPENDS_ON"
```

When `start_node_id` and `end_node_id` are Atlassian Resource Identifiers (ARIs), the composite ID contains colons. The provider parses the ID by recognizing the ARI segment pattern (`ari:cloud:compass:...`) so that each component ID and `relationship_type` are correctly split. Use the exact format above (both IDs and type joined by colons; the whole string may need to be quoted in the shell).

## Behavior notes

- **Already exists:** If you create a relationship that already exists in Compass (e.g. created outside Terraform or by another run), the API may return an error. The provider treats this as success: it sets the composite ID and attributes in state and runs a read to adopt the existing relationship, so `terraform apply` does not fail.
- **No update:** All arguments are `ForceNew`. To change the relationship (e.g. different type or nodes), delete the resource and create a new one.

## Direction semantics

* **DEPENDS_ON**: `start_node_id` depends on `end_node_id` (start → end).
* **CHILD_OF**: `start_node_id` is a child of `end_node_id` (start → end).

Changing any of `start_node_id`, `end_node_id`, or `relationship_type` forces recreation of the resource (ForceNew).
