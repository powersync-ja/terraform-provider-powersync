# Read an existing PowerSync instance that was created outside of Terraform
# (e.g. via the dashboard), or round-trip attributes like `instance_url` from a
# managed instance into outputs.
data "powersync_instance" "main" {
  org_id     = "64b3f8e1a2c4d5e6f7080912"
  project_id = "699ef9c371c56d0007320543"
  id         = "6a05387093eae275999af147"
}

output "instance_url" {
  value = data.powersync_instance.main.instance_url
}

output "instance_status" {
  value = data.powersync_instance.main.status
}
