variable "compass_email" {
  description = "Email address of your Atlassian account"
  type        = string
}

variable "compass_api_token" {
  description = "Atlassian API token for Compass authentication. Get it from https://id.atlassian.com/manage/api-tokens"
  type        = string
  sensitive   = true
}

variable "compass_tenant" {
  description = "Tenant name for automatic cloud_id detection (e.g., 'your_site_name' for your_site_name.atlassian.net)"
  type        = string
}
