# ── Organization ────────────────────────────────────────────────────────────────

output "org_id" {
  value = data.powersync_organization.main.id
}

output "org_name" {
  value = data.powersync_organization.main.name
}

# ── Project ─────────────────────────────────────────────────────────────────────

output "project_id" {
  value = data.powersync_project.terraform_project.id
}

output "project_name" {
  value = data.powersync_project.terraform_project.name
}

output "project_default_region" {
  value = data.powersync_project.terraform_project.default_region
}

output "project_vcs_mode" {
  value = data.powersync_project.terraform_project.vcs_mode
}

output "project_trial" {
  value = data.powersync_project.terraform_project.trial
}

output "project_locked" {
  value = data.powersync_project.terraform_project.locked
}

# ── Projects (list data source) ────────────────────────────────────────────────

output "projects_total" {
  value = data.powersync_projects.all.total
}

output "projects_count" {
  value = length(data.powersync_projects.all.projects)
}

output "projects_names" {
  value = [for p in data.powersync_projects.all.projects : p.name]
}

output "projects_all" {
  value = data.powersync_projects.all.projects
}

# ── Instance ─────────────────────────────────────────────────────────────────────

output "instance_id" {
  value = powersync_instance.terraform_instance.id
}

output "instance_name" {
  value = powersync_instance.terraform_instance.name
}

output "instance_region" {
  value = powersync_instance.terraform_instance.region
}

output "instance_status" {
  value = powersync_instance.terraform_instance.status
}

output "instance_provisioned" {
  value = powersync_instance.terraform_instance.provisioned
}

output "instance_url" {
  value = powersync_instance.terraform_instance.instance_url
}

output "instance_sync_config_content" {
  value = powersync_instance.terraform_instance.sync_config_content
}

output "instance_operations" {
  value = powersync_instance.terraform_instance.operations
}

output "instance_replication_connections" {
  value = powersync_instance.terraform_instance.replication_connection
}

output "instance_client_auth" {
  value = powersync_instance.terraform_instance.client_auth
}

# ── Instance (data source) ──────────────────────────────────────────────────────

output "instance_ds_id" {
  value = data.powersync_instance.existing.id
}

output "instance_ds_name" {
  value = data.powersync_instance.existing.name
}

output "instance_ds_region" {
  value = data.powersync_instance.existing.region
}

output "instance_ds_status" {
  value = data.powersync_instance.existing.status
}

output "instance_ds_provisioned" {
  value = data.powersync_instance.existing.provisioned
}

output "instance_ds_url" {
  value = data.powersync_instance.existing.instance_url
}

output "instance_ds_sync_config" {
  value = data.powersync_instance.existing.sync_config_content
}

output "instance_ds_operations" {
  value = data.powersync_instance.existing.operations
}

output "instance_ds_replication_connections" {
  value = data.powersync_instance.existing.replication_connection
}

output "instance_ds_client_auth" {
  value = data.powersync_instance.existing.client_auth
}