terraform {
  required_providers {
    powersync = {
      source  = "powersync/powersync"
      version = "~> 0.1.0"
    }
  }
}

provider "powersync" {
  # admin_token is picked up from the PS_PAT_TOKEN environment variable (recommended).
  # It can also be inlined — less secure, but if inlined via a variable the value
  # itself can still be passed securely through Terraform's own env vars
  # (e.g. TF_VAR_ps_admin_token).
  # admin_token  = var.admin_token
  accounts_url   = "https://accounts.staging.powersync.com"
  management_url = "https://powersync-api.staging.journeyapps.com"
}

locals {
  org_id = "69e1ded296488e0007395292"

  # Supabase project we replicate from.
  supabase_db_host = "db.erzuanfjiinltpklcunu.supabase.co"
  supabase_db_name = "postgres"
  supabase_db_user = "powersync_role"
}

variable "replication_password" {
  description = "Password for the powersync_role on the Supabase Postgres. Set via TF_VAR_replication_password."
  type        = string
  sensitive   = true
}

# ── Organization ───────────────────────────────────────────────────────────────

data "powersync_organization" "main" {
  id = local.org_id
}

# ── Project (managed) ──────────────────────────────────────────────────────────

resource "powersync_project" "main" {
  org_id = data.powersync_organization.main.id
  name   = "Terraform Project"
  region = "eu"

  # Uncomment to allow destroy when un-managed instances exist under this project.
  # force_destroy = true
}

# ── Instance (managed) ─────────────────────────────────────────────────────────

resource "powersync_instance" "main" {
  org_id     = data.powersync_organization.main.id
  project_id = powersync_project.main.id
  name       = "terraform-instance"
  region     = "staging" # staging env uses "staging" region; production uses "eu", "us", etc.

  replication_connection {
    type     = "postgresql"
    name     = "supabase-main"
    hostname = local.supabase_db_host
    port     = 5432
    username = local.supabase_db_user
    password = var.replication_password
    database = local.supabase_db_name
    sslmode  = "verify-full"
    # cacert omitted: PowerSync ships Supabase's CA cert by default with verify-full.
  }

  client_auth {
    supabase               = true
    allow_temporary_tokens = true
  }

  sync_config_content = <<-YAML
    config:
      edition: 3
    streams:
      todos:
        auto_subscribe: true
        query: SELECT * FROM todos
  YAML
}

# ── Data sources (round-tripping the managed resources) ────────────────────────

data "powersync_project" "main" {
  org_id = data.powersync_organization.main.id
  id     = powersync_project.main.id
}

data "powersync_projects" "all" {
  org_id = data.powersync_organization.main.id
}

data "powersync_instance" "main" {
  org_id     = data.powersync_organization.main.id
  project_id = powersync_project.main.id
  id         = powersync_instance.main.id
}

data "powersync_instances" "all" {
  org_id     = data.powersync_organization.main.id
  project_id = powersync_project.main.id
}
