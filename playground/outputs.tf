# ── Organization ────────────────────────────────────────────────────────────────

output "org_id" {
  value = data.powersync_organization.main.id
}

output "org_name" {
  value = data.powersync_organization.main.name
}

# ── Project (managed) ──────────────────────────────────────────────────────────

output "project_id" {
  value = powersync_project.main.id
}

output "project_name" {
  value = powersync_project.main.name
}

output "project_region" {
  value = powersync_project.main.region
}

# ── Instance (managed) ─────────────────────────────────────────────────────────

output "instance_id" {
  value = powersync_instance.main.id
}

output "instance_name" {
  value = powersync_instance.main.name
}

output "instance_region" {
  value = powersync_instance.main.region
}

output "instance_status" {
  value = powersync_instance.main.status
}

output "instance_provisioned" {
  value = powersync_instance.main.provisioned
}

output "instance_url" {
  value = powersync_instance.main.instance_url
}

output "instance_sync_config_content" {
  value = powersync_instance.main.sync_config_content
}

output "instance_replication_connections" {
  value = powersync_instance.main.replication_connection
  sensitive = true
}

output "instance_client_auth" {
  value = powersync_instance.main.client_auth
}

