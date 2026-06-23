---
page_title: "2. Connecting a source database"
subcategory: "Guides"
description: |-
  Configure the replication_connection block for each supported source database — PostgreSQL, MongoDB, MySQL, and SQL Server — and understand what each one needs.
---

# Connecting a source database

A PowerSync instance replicates from exactly one source database, described by the `replication_connection` block on `powersync_instance`. This guide covers what that block looks like for each supported database type and the connection-level nuances that differ between them. For the end-to-end flow (token, project, apply), start with the [Getting Started guide](getting-started.md).

PowerSync supports four source database types: `postgresql`, `mongodb`, `mysql`, and `mssql`. MySQL and SQL Server are currently in beta.

This guide is about the *connection* — the fields Terraform sends to PowerSync. Each database also has **upstream prerequisites** (replication has to be enabled on the database itself) that live outside Terraform; those are linked per-section below.

## The `replication_connection` block

A few rules apply regardless of database type:

- **One connection per instance.** At most one `replication_connection` block is supported.
- **`uri` *or* discrete fields, never both.** Provide either a single `uri` (e.g. `postgresql://user:pass@host:5432/db`) or the separate `hostname`/`port`/`username`/`password`/`database` fields. They are mutually exclusive.
- **Use a dedicated, least-privilege user.** `username` should be a replication-only user with the minimum privileges PowerSync needs — not an admin or superuser account. Keep its `password` in a Terraform variable marked `sensitive`; PowerSync stores it server-side as a secret and redacts it from plan/apply output.
- **`name`** is a display label shown in the dashboard and has no functional effect. **`tag`** (defaults to `default`) is how the sync config references this connection.

### Which fields apply to which database

| Field | PostgreSQL | MongoDB | MySQL | SQL Server |
|---|:---:|:---:|:---:|:---:|
| `hostname` / `port` / `username` / `password` (or `uri`) | yes | yes | yes | yes |
| `database` | yes | yes | yes | no |
| `sslmode` | yes | no | yes | no |
| `cacert` | yes | no | yes | no |
| `client_certificate` / `client_private_key` (mTLS) | yes | no | yes | no |
| `post_images` | no | yes | no | no |
| `schema` | no | no | no | yes |

## TLS (PostgreSQL and MySQL)

PowerSync accepts only the two strong TLS modes — weaker ones (`require`, `prefer`, `disable`) are rejected:

- `sslmode = "verify-full"` (default) — verifies the certificate chain **and** hostname.
- `sslmode = "verify-ca"` — verifies the certificate chain only.

For three managed **PostgreSQL** providers — **Supabase**, **AWS RDS**, and **Azure Postgres** — PowerSync bundles the CA certificate, so `verify-full` works against them without supplying anything extra. For every other case (other Postgres hosts such as Google Cloud SQL, Neon, or PlanetScale, self-hosted databases, and MySQL) you supply the server's CA via `cacert` (PEM-encoded).

For mutual TLS, supply `client_certificate` and `client_private_key` together (both PEM-encoded; the key is stored server-side as a secret).

## PostgreSQL

**Upstream prerequisite:** logical replication enabled, plus a publication scoped to the tables you want synced. See [PowerSync's PostgreSQL setup docs](https://docs.powersync.com/configuration/source-db/setup#postgres) — they include provider-specific notes for Supabase, AWS RDS, Azure, Neon, and others.

```hcl
replication_connection {
  type     = "postgresql"
  name     = "primary"
  hostname = "db.example.com"
  port     = 5432
  database = "postgres"
  username = "powersync"
  password = var.replication_password
  sslmode  = "verify-full"
}
```

## MongoDB

**Upstream prerequisite:** change streams (MongoDB 3.6+) and a user with the right read permissions. See [PowerSync's MongoDB setup docs](https://docs.powersync.com/configuration/source-db/setup#mongodb).

`post_images` controls the change-stream `fullDocument` mode: `off` (document key only), `auto_configure` (PowerSync enables `changeStreamPreAndPostImages` on collections for you), or `read_only` (assume it's already configured upstream).

For MongoDB Atlas `mongodb+srv://` connection strings (which carry no port), use the `uri` field instead of `hostname`/`port`:

```hcl
replication_connection {
  type        = "mongodb"
  name        = "primary"
  uri         = "mongodb+srv://powersync:${var.replication_password}@cluster0.abcd.mongodb.net/myapp"
  post_images = "auto_configure"
}
```

## MySQL (beta)

**Upstream prerequisite:** binary logging enabled with GTID replication configured. See [PowerSync's MySQL setup docs](https://docs.powersync.com/configuration/source-db/setup#mysql).

TLS works the same as PostgreSQL (`sslmode`, optional `cacert`, optional mTLS).

```hcl
replication_connection {
  type     = "mysql"
  name     = "primary"
  hostname = "db.example.com"
  port     = 3306
  database = "myapp"
  username = "powersync"
  password = var.replication_password
  sslmode  = "verify-full"
}
```

## SQL Server (beta)

**Upstream prerequisite:** Change Data Capture (CDC) enabled on the database. See [PowerSync's SQL Server setup docs](https://docs.powersync.com/configuration/source-db/setup#sql-server).

Use `schema` to set the default schema for replicated tables (e.g. `dbo`).

```hcl
replication_connection {
  type     = "mssql"
  name     = "primary"
  hostname = "db.example.com"
  port     = 1433
  database = "myapp"
  schema   = "dbo"
  username = "powersync"
  password = var.replication_password
}
```
