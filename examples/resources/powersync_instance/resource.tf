data "powersync_organization" "main" {
  id = "64b3f8e1a2c4d5e6f7080912"
}

resource "powersync_project" "main" {
  org_id = data.powersync_organization.main.id
  name   = "My Project"
  region = "eu"
}

resource "powersync_instance" "production" {
  org_id     = data.powersync_organization.main.id
  project_id = powersync_project.main.id
  name       = "production"
  # region defaults to the project's region when omitted.

  # Replicate from a Supabase Postgres. PowerSync ships Supabase's CA cert by
  # default, so verify-full works without a cacert. See PowerSync's connection
  # docs for MongoDB, MySQL, and MSSQL examples.
  replication_connection {
    type     = "postgresql"
    name     = "supabase-main"
    hostname = "db.abcdefghijklmnopqrst.supabase.co"
    port     = 5432
    database = "postgres"
    username = "powersync_role"
    password = var.replication_password
    sslmode  = "verify-full"
  }

  # Validate client JWTs issued by Supabase Auth (asymmetric / new projects).
  # For legacy HS256 Supabase projects, see the inline-JWKS variant in the
  # connecting-supabase guide.
  client_auth {
    supabase               = true
    allow_temporary_tokens = false
  }

  # Sync config — describes what each client gets and how it's partitioned.
  # https://docs.powersync.com/sync/overview
  sync_config_content = <<-YAML
    config:
      edition: 3
    streams:
      todos:
        auto_subscribe: true
        query: SELECT * FROM todos WHERE user_id = request.user_id()
  YAML
}

variable "replication_password" {
  description = "Password for the powersync_role on the source database. Set via TF_VAR_replication_password."
  type        = string
  sensitive   = true
}

output "instance_url" {
  value = powersync_instance.production.instance_url
}
