# Look up a PowerSync organization by its ID. The organization ID is visible in
# the PowerSync dashboard URL: https://www.powersync.com/dashboard/orgs/<org_id>.
data "powersync_organization" "main" {
  id = "64b3f8e1a2c4d5e6f7080912"
}

output "org_name" {
  value = data.powersync_organization.main.name
}
