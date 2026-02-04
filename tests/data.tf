# All component types
data "compass_component_types" "all" {}

output "component_types" {
  description = "All available Compass component types (id and name) for the site"
  value       = data.compass_component_types.all.types
}

# Component type by id
data "compass_component_types" "by_id" {
  id = "SERVICE"
}

output "by_id" {
  description = "Component type(s) filtered by id (e.g. SERVICE or ARI)"
  value       = data.compass_component_types.by_id.types
}

# Component type by name
data "compass_component_types" "by_name" {
  name = "Domain"
}

output "by_name" {
  description = "Component type(s) filtered by human-readable name (e.g. Domain)"
  value       = data.compass_component_types.by_name.types
}