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

# ── Instance ─────────────────────────────────────────────────────────────────────

output "instance_id" {
  value = powersync_instance.terraform_instance.id
}

output "instance_name" {
  value = powersync_instance.terraform_instance.name
}