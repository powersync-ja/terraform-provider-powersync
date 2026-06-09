---
page_title: "2. Connecting Supabase"
subcategory: "Guides"
description: |-
  Set up a Supabase Postgres as the replication source for a PowerSync instance — DB role, publication, and JWT validation.
---

# Connecting Supabase

This guide covers the Supabase-specific setup that has to happen *outside* of Terraform before `terraform apply` will produce a working PowerSync instance backed by a Supabase Postgres. The PowerSync side itself is the same as any other source DB — see the [Getting Started guide](getting-started.md) for that.

The setup has two parts:

1. **Database side**: a dedicated replication role + a publication scoped to the tables you want PowerSync to sync.
2. **Auth**: configuring PowerSync to validate the JWTs Supabase issues to your client app.

## 1. Database setup

Create a dedicated `powersync_role` with only the privileges PowerSync needs. Connect to your Supabase Postgres as the `postgres` user (e.g. from the SQL editor in the Supabase dashboard or via `psql`) and run:

```sql
-- A throwaway, rotatable password — set it via your secrets manager.
CREATE ROLE powersync_role WITH REPLICATION LOGIN PASSWORD '<a-strong-password>';

-- The publication PowerSync will subscribe to. Replace `todos` with the tables
-- you actually want replicated. You can add or remove tables here without
-- redeploying the PowerSync instance.
CREATE PUBLICATION powersync FOR TABLE public.todos;

-- powersync_role needs read access to the tables in the publication.
GRANT USAGE ON SCHEMA public TO powersync_role;
GRANT SELECT ON public.todos TO powersync_role;
```

Why a dedicated role?

- Principle of least privilege: PowerSync only needs `REPLICATION` plus `SELECT` on the published tables. The `postgres` superuser is far more access than required.
- Easy rotation: rotate the `powersync_role` password without touching anything else.
- Easy revocation: drop the role to kill replication if something goes wrong.

## 2. Auth — JWT validation

Supabase issues JWTs to your client app on login. PowerSync needs to validate those JWTs to authorize sync requests. Supabase has two signing-key modes:

- **New asymmetric keys (RS256)** — *supported*, and the default for new Supabase projects. PowerSync auto-detects your Supabase project from the replication connection string and discovers the JWKS endpoint (`https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json`) and `authenticated` audience for you. Just set:

  ```hcl
  client_auth {
    supabase = true
  }
  ```

- **Legacy HS256 keys (symmetric secret)** — *on the roadmap, not yet supported*. If **Settings → API → JWT Settings** shows a single "JWT Secret" string, your project is still on legacy keys; follow Supabase's [Rotate to asymmetric JWTs](https://supabase.com/blog/jwt-signing-keys) migration in the meantime.

## 3. Terraform configuration

Put it all together. The relevant fragment of your Terraform config:

```hcl
resource "powersync_instance" "production" {
  org_id     = data.powersync_organization.main.id
  project_id = powersync_project.main.id
  name       = "production"

  replication_connection {
    type     = "postgresql"
    name     = "supabase-main"
    hostname = "db.<your-project-ref>.supabase.co"
    port     = 5432
    database = "postgres"
    username = "powersync_role"
    password = var.replication_password
    sslmode  = "verify-full"

    # No cacert needed: PowerSync ships Supabase's CA cert by default.
  }

  client_auth {
    supabase               = true
    allow_temporary_tokens = false
  }

  sync_config_content = file("${path.module}/sync-rules.yaml")
}

variable "replication_password" {
  description = "Password for powersync_role on the Supabase Postgres."
  type        = string
  sensitive   = true
}
```

Apply it:

```sh
export PS_PAT_TOKEN="jpt_..."
export TF_VAR_replication_password="<the password you set on powersync_role>"
terraform apply
```

If `apply` reports `connection test failed`, check that:

1. `TF_VAR_replication_password` matches what you actually set on `powersync_role` (re-`export` if you've regenerated it since).
2. The `powersync` publication exists and includes the tables you reference in your sync config.
