---
page_title: "1. Getting Started"
subcategory: "Guides"
description: |-
  Walk through provisioning a PowerSync instance from an empty Terraform project to a deployed, sync-ready environment.
---

# Getting Started with the PowerSync provider

This guide assumes you have a Terraform install, a PowerSync account, and a PostgreSQL/MongoDB/MySQL/MSSQL database you want PowerSync to replicate from. By the end of it, you'll have a deployed PowerSync instance configured to replicate from your source database and validate JWTs from your client app.

If you've never used Terraform before: Terraform is a tool that turns a description of your infrastructure (written in a configuration language called HCL) into the actual cloud resources that match that description. You write what you want, run `terraform apply`, and Terraform calls the relevant APIs to make it so. The PowerSync provider is the piece that teaches Terraform how to talk to PowerSync's API.

## 1. Generate a personal access token

1. Open your PowerSync dashboard.
2. Go to **Account Settings → Personal Access Tokens** and click **Create new token**.
3. Copy the token. Treat it like a password — it grants full account access.
4. Export it:

   ```sh
   export PS_PAT_TOKEN="jpt_..."
   ```

## 2. Find your organization ID

The provider needs to know which organization to create resources in. The organization ID is in the dashboard URL — when you're on the org's home page, the URL looks like `https://www.powersync.com/dashboard/orgs/64b3f8e1a2c4d5e6f7080912/...`. The hex segment is your org ID.

## 3. Write the configuration

Create a new directory for the project and a `main.tf` file inside it:

```hcl
terraform {
  required_providers {
    powersync = {
      source  = "powersync-ja/powersync"
      version = "~> 0.1"
    }
  }
}

provider "powersync" {}

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

  replication_connection {
    type     = "postgresql"
    name     = "main-db"
    hostname = "db.example.com"
    port     = 5432
    database = "postgres"
    username = "powersync_role"
    password = var.replication_password
    sslmode  = "verify-full"
  }

  client_auth {
    jwks_uri               = "https://auth.example.com/.well-known/jwks.json"
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

variable "replication_password" {
  type      = string
  sensitive = true
}
```

A few things to point out in this config:

- The `data "powersync_organization"` block is a *data source* — it reads an existing organization. We don't manage orgs as Terraform resources; they're created when you sign up.
- `powersync_project` and `powersync_instance` are *resources* — Terraform creates them, tracks them in state, and tears them down on `terraform destroy`.
- `replication_password` is a Terraform variable. The actual value is passed in at apply-time via the environment so it never appears in plain text in the config or state. PowerSync stores it server-side in its secrets store.

## 4. Apply

```sh
export TF_VAR_replication_password="<the password of powersync_role on your DB>"
terraform init
terraform plan
terraform apply
```

`terraform init` downloads the provider. `terraform plan` shows you what Terraform is about to do — read it before approving. `terraform apply` actually creates the project, calls `instances/create`, then `instances/deploy`, then polls until the deploy completes (~2–3 minutes).

When `apply` finishes, the instance is live. Inspect:

```sh
terraform output                          # everything
terraform show                            # full state
terraform state list                      # every managed resource
```

## 5. Iterate

Change anything — instance name, sync config, replication connection — and re-run `terraform plan` then `apply`. Every update goes through PowerSync's deploy endpoint and triggers a full redeploy (~2–3 minutes), which is a constraint of the management API: there is no separate "rename" or "patch" endpoint, so any change is treated as a redeploy. The instance ID stays the same; only the configuration is updated in place.

To force a recreate (e.g. moving regions, which is `ForceNew` on the instance), Terraform plans a destroy followed by a create automatically.

## 6. Tear down

```sh
terraform destroy
```

This destroys both the instance and the project. If the project owns unmanaged instances (created outside of Terraform), the destroy will fail — set `force_destroy = true` on `powersync_project` to override.

## Where to go next

- [Connecting Supabase](connecting-supabase.md) — Supabase-specific gotchas (publication setup, JWT modes, IPv4 add-on).
- [`powersync_instance` reference](../resources/instance.md) — full schema for the instance resource.
- [PowerSync sync config docs](https://docs.powersync.com/sync/overview) — what to put in `sync_config_content`.
