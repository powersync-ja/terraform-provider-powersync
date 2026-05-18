data "powersync_organization" "main" {
  id = "64b3f8e1a2c4d5e6f7080912"
}

# List every project under the organization.
data "powersync_projects" "all" {
  org_id = data.powersync_organization.main.id
}

output "project_count" {
  value = length(data.powersync_projects.all.projects)
}

output "project_names" {
  value = [for p in data.powersync_projects.all.projects : p.name]
}
