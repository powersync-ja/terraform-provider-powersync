data "powersync_organization" "main" {
  id = "64b3f8e1a2c4d5e6f7080912"
}

resource "powersync_project" "main" {
  org_id = data.powersync_organization.main.id
  name   = "My Project"
  region = "eu"

  # Allow `terraform destroy` even when un-managed instances exist under this
  # project. Without this flag the destroy fails with a guard error.
  # force_destroy = true
}

output "project_id" {
  value = powersync_project.main.id
}
