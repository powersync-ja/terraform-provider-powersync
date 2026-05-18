data "powersync_organization" "main" {
  id = "64b3f8e1a2c4d5e6f7080912"
}

# Look up a project by name within an org.
data "powersync_project" "by_name" {
  org_id = data.powersync_organization.main.id
  name   = "My Project"
}

# Or by ID.
data "powersync_project" "by_id" {
  org_id = data.powersync_organization.main.id
  id     = "699ef9c371c56d0007320543"
}

output "project_id" {
  value = data.powersync_project.by_name.id
}

output "default_region" {
  value = data.powersync_project.by_name.default_region
}
