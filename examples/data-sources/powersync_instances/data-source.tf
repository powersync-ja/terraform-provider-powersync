# List every instance under a project. Useful for inventory/auditing or to look
# up an instance ID by name without hard-coding it.
data "powersync_instances" "all" {
  org_id     = "64b3f8e1a2c4d5e6f7080912"
  project_id = "699ef9c371c56d0007320543"
}

output "instance_count" {
  value = length(data.powersync_instances.all.instances)
}

output "instance_summary" {
  value = [for i in data.powersync_instances.all.instances : {
    id         = i.id
    name       = i.name
    deployable = i.deployable
    has_config = i.has_config
  }]
}
