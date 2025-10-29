# Component
output "component_id" {
  description = "ID of the created component"
  value       = compass_component.example.id
}

output "component_name" {
  description = "Name of the created component"
  value       = compass_component.example.name
}

output "component_description" {
  description = "Description of the created component"
  value       = compass_component.example.description
}

output "component_type" {
  description = "Type of the created component"
  value       = compass_component.example.type
}

# Component Link
output "component_link_id" {
  description = "ID of the created component link"
  value       = compass_component_link.repository.id
}

output "component_link_name" {
  description = "Name of the created component link"
  value       = compass_component_link.repository.name
}

output "component_link_url" {
  description = "URL of the created component link"
  value       = compass_component_link.repository.url
}

output "component_link_type" {
  description = "Type of the created component link"
  value       = compass_component_link.repository.type
}