# compass_component_labels

Manages the set of labels on a Compass component. One resource per component: the resource represents the full list of labels for that component. On update, the provider computes the diff between desired and current labels and calls the Compass API to add new labels and remove labels no longer in the list.

## Example Usage

### Basic labels

```hcl
resource "compass_component" "example" {
  name = "My Service"
  type = "SERVICE"
}

resource "compass_component_labels" "example" {
  component_id = compass_component.example.id
  labels       = ["production", "dotnet"]
}
```

### Single label

```hcl
resource "compass_component_labels" "example" {
  component_id = compass_component.example.id
  labels       = ["team-backend"]
}
```

### With explicit cloud_id

```hcl
resource "compass_component_labels" "example" {
  component_id = compass_component.example.id
  cloud_id     = "a12345-f505-432c-99ff-5d58963aa42c"
  labels       = ["production", "critical"]
}
```

## Argument Reference

The following arguments are supported:

* `component_id` - (Required, ForceNew) ID of the Compass component to manage labels for. Use the full ARI from `compass_component.id`.
* `labels` - (Required) List of label names to set on the component. Treated as a set; order is normalized and duplicates are ignored. At least one label is required on create.
* `cloud_id` - (Optional, Computed, ForceNew) Cloud ID of the Atlassian site. If not provided, will be automatically detected from the `tenant` configured in the provider.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Same as `component_id` (one labels resource per component).
* `cloud_id` - Cloud ID (computed if not provided explicitly).

## Import

Component labels can be imported by component ID:

```bash
terraform import compass_component_labels.example "ari:cloud:compass:...:component/..."
```

Optional format with cloud_id: `component_id:cloud_id` or `component_id/cloud_id`. The resource will read current labels from the API after import.

## Update behavior

* Changing `labels`: the provider computes which labels to add and which to remove, then calls `addComponentLabels` for new ones and `removeComponentLabels` for removed ones. Remove is applied first, then add.
* Reordering the list in config does not cause any API calls; labels are compared as a set.
* `component_id` and `cloud_id` cannot be changed after creation (ForceNew).

## Notes

* There should be at most one `compass_component_labels` resource per component. Managing the same component from multiple resources will cause conflicting updates.
* Labels are read from the Compass component's `labels` field. If the component is deleted, the next read will remove the resource from state.
* The label `synced-with-jsm` is added automatically by Compass when a component is synced with Jira. The provider never removes it: it is excluded from remove operations and from state so that Terraform does not show perpetual plan drift. You do not need to add it to `labels` in config.
