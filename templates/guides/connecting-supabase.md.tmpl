---
page_title: "2. Connecting Supabase"
subcategory: "Guides"
description: |-
  Set up a Supabase Postgres as the replication source for a PowerSync instance — DB role, publication, network reachability, and JWT validation.
---

# Connecting Supabase

This guide covers the Supabase-specific setup that has to happen *outside* of Terraform before `terraform apply` will produce a working PowerSync instance backed by a Supabase Postgres. The PowerSync side itself is the same as any other source DB — see the [Getting Started guide](getting-started.md) for that.

The setup has three parts:

1. **Database side**: a dedicated replication role + a publication scoped to the tables you want PowerSync to sync.
2. **Network**: making the database actually reachable from PowerSync's network.
3. **Auth**: configuring PowerSync to validate the JWTs Supabase issues to your client app.

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

## 2. Network reachability

> **Important.** As of 2024, Supabase moved direct Postgres connections (`db.<ref>.supabase.co`) to IPv6-only for projects without the IPv4 add-on. PowerSync Cloud's egress is IPv4. If you try to connect from PowerSync to a free-tier Supabase project, the connection test will fail with `ECONNREFUSED` against an IPv6 address — *before* any authentication happens.

You have two options:

- **Enable Supabase's IPv4 add-on** (recommended; matches what PowerSync's official docs assume). Go to your project's **Settings → Add-ons → Dedicated IPv4 Address** and toggle it on. The direct DB endpoint then resolves to IPv4 and PowerSync can reach it. This is a small monthly add-on cost.

- **Use Supavisor (Supabase's connection pooler, dual-stack)** as the replication target instead of the direct endpoint. Use the *session-mode* pooler (port 5432, not 6543) — transaction-mode does not support logical replication. The hostname and username format are different from the direct connection; check the **Connection string** section of your Supabase project's database settings. Note that logical replication through the pooler is not formally supported by Supabase for every role, so test carefully.

## 3. Auth — JWT validation

Supabase issues JWTs to your client app on login. PowerSync needs to validate those JWTs to authorize sync requests. Two modes, depending on when your Supabase project was created:

- **Asymmetric keys (default for new projects, ~late 2024 onward)**: Supabase publishes a JWKS at `https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json`. PowerSync knows how to discover and use that endpoint when you set:

  ```hcl
  client_auth {
    supabase = true
  }
  ```

- **Legacy HS256 (older projects)**: the JWT is signed with a symmetric secret stored in your Supabase project settings. PowerSync needs that secret to validate tokens. Support for this mode requires the `jwks` inline-keys configuration on the instance — see the `powersync_instance` resource docs for the JWK structure.

Check which mode you're in: in your Supabase dashboard go to **Settings → API → JWT Settings**. If it shows a single "JWT Secret" string, you're on HS256. If it shows multiple signing keys with `kid`s and curve/algorithm info, you're on the new asymmetric setup.

## 4. Terraform configuration

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
export PS_PAT_TOKEN="pst_..."
export TF_VAR_replication_password="<the password you set on powersync_role>"
terraform apply
```

If `apply` reports `connection test failed`, check that:

1. `TF_VAR_replication_password` matches what you actually set on `powersync_role` (re-`export` if you've regenerated it since).
2. The Supabase project has IPv4 reachable (Add-on enabled, or pooler used).
3. The `powersync` publication exists and includes the tables you reference in your sync config.

If `apply` reports `Failed to deploy instance: ...`, the connection test passed but the deploy itself errored — most often a malformed sync config YAML or a region mismatch. The error message usually points at the offending field.
