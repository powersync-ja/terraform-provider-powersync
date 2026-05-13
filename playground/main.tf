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
  project_id = "69fa95159e449e0007987a88"
}

# ── Organization ────────────────────────────────────────────────────────────────

data "powersync_organization" "main" {
  id = local.org_id
}

# ── Project ─────────────────────────────────────────────────────────────────────

data "powersync_project" "terraform_project" {
  org_id = data.powersync_organization.main.id
  id     = local.project_id
}

# ── Projects (list data source) ────────────────────────────────────────────────

data "powersync_projects" "all" {
  org_id = data.powersync_organization.main.id
}

# ── Instance ─────────────────────────────────────────────────────────────────────

resource "powersync_instance" "terraform_instance" {
  org_id     = data.powersync_organization.main.id
  project_id = data.powersync_project.terraform_project.id
  name   = "terraform-instance"
  region = "staging" # staging env uses "staging" region; production uses "eu", "us", etc.

  # replication_connection {
  #   type     = "postgresql"
  #   name     = "main-db"
  #   hostname = "db.example.com"
  #   port     = 5432
  #   username = "powersync"
  #   password = "changeme"
  #   database = "mydb"
  #   sslmode  = "verify-full"
  # }

  client_auth {
    allow_temporary_tokens = true
  }

  sync_config_content = <<-YAML
    bucket_definitions:
      - name: all
        parameters: []
        data:
          - table: todos
  YAML
}

# ── Instance (data source) ──────────────────────────────────────────────────────

data "powersync_instance" "existing" {
  org_id     = data.powersync_organization.main.id
  project_id = data.powersync_project.terraform_project.id
  id         = "69fa951654d621dd291948ea"
}