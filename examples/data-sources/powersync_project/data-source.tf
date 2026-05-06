data "powersync_organization" "main" {
  name = "My Organization"
}

# Look up a project by name within an org
data "powersync_project" "main" {
  org_id = data.powersync_organization.main.id
  name   = "My Project"
}

# Or by ID:
# data "powersync_project" "main" {
#   org_id = data.powersync_organization.main.id
#   id     = "699ef9c371c56d0007320543"
# }

output "project_id" {
  value = data.powersync_project.main.id
}
