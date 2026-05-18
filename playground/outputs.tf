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

# ── Project (data source — round-trip) ─────────────────────────────────────────

output "project_ds_name" {
  value = data.powersync_project.main.name
}

output "project_ds_default_region" {
  value = data.powersync_project.main.default_region
}

# ── Projects (list data source) ────────────────────────────────────────────────

output "projects_count" {
  value = length(data.powersync_projects.all.projects)
}

output "projects_names" {
  value = [for p in data.powersync_projects.all.projects : p.name]
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

output "instance_operations" {
  value = powersync_instance.main.operations
}

output "instance_replication_connections" {
  value = powersync_instance.main.replication_connection
  sensitive = true
}

output "instance_client_auth" {
  value = powersync_instance.main.client_auth
}

# ── Instance (data source — round-trip) ────────────────────────────────────────

output "instance_ds_id" {
  value = data.powersync_instance.main.id
}

output "instance_ds_name" {
  value = data.powersync_instance.main.name
}

output "instance_ds_region" {
  value = data.powersync_instance.main.region
}

output "instance_ds_status" {
  value = data.powersync_instance.main.status
}

output "instance_ds_provisioned" {
  value = data.powersync_instance.main.provisioned
}

output "instance_ds_url" {
  value = data.powersync_instance.main.instance_url
}

output "instance_ds_sync_config" {
  value = data.powersync_instance.main.sync_config_content
}

output "instance_ds_operations" {
  value = data.powersync_instance.main.operations
}

output "instance_ds_replication_connections" {
  value = data.powersync_instance.main.replication_connection
}

output "instance_ds_client_auth" {
  value = data.powersync_instance.main.client_auth
}

# ── Instances (list data source) ───────────────────────────────────────────────

output "instances_count" {
  value = length(data.powersync_instances.all.instances)
}

output "instances_names" {
  value = [for i in data.powersync_instances.all.instances : i.name]
}

output "instances_all" {
  value = data.powersync_instances.all.instances
}
