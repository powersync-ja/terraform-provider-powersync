# Terraform Provider for PowerSync

[![Terraform Registry](https://img.shields.io/badge/registry-powersync--ja%2Fpowersync-623CE4)](https://registry.terraform.io/providers/powersync-ja/powersync/latest/docs)

The official [PowerSync](https://www.powersync.com/) Terraform provider. Manage your PowerSync Cloud projects and instances as Infrastructure-as-Code, alongside the rest of your infrastructure.

## What's this for?

[Terraform](https://www.terraform.io/) is a HashiCorp tool for managing infrastructure as code. Instead of manually creating cloud resources, DNS records, IAM roles, databases, and other infrastructure, you define the desired state in HCL and run `terraform apply`. Terraform then determines what needs to be created, updated, or removed to match your configuration.

This provider integrates Terraform with the PowerSync cloud, allowing you to manage PowerSync resources directly through Terraform. With it, you can:

- **Provision a PowerSync alongside the rest of your infrastructure**: using the same terraform apply workflow that creates your database, networking, and supporting services.
- **Keep `dev`/`staging`/`prod` environments consistently**: define them once, parameterize it with variables instead of configuring everything manually in the dashboard.
- **Review infrastructure changes in PRs**: changes to sync config, replication connections, JWKS settings show up as Terraform plan diffs and Git.
- **Recreate environments from scratch when needed**: using version-controlled Terraform configuration as a repeatable source of truth.

If you've never used Terraform before, check the [Getting Started guide](docs/guides/getting-started.md) for more information.

## Quickstart

```hcl
terraform {
  required_providers {
    powersync = {
      source  = "powersync-ja/powersync"
      version = "~> 0.1.0"
    }
  }
}

# admin_token is picked up from the PS_PAT_TOKEN environment variable (recommended).
# It can also be inlined — less secure, but if inlined via a variable the value
# itself can still be passed securely through Terraform's own env vars
# (e.g. TF_VAR_ps_admin_token).
# admin_token  = var.admin_token
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
    name     = "main"
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

  sync_config_content = file("${path.module}/sync-config.yaml")
}

variable "replication_password" {
  type      = string
  sensitive = true
}
```

```sh
export PS_PAT_TOKEN="jpt_..."                       # generate from PowerSync dashboard
export TF_VAR_replication_password="..."
terraform init
terraform apply
```

## Documentation

Generated documentation is hosted on the Terraform Registry:

- [Provider overview](https://registry.terraform.io/providers/powersync-ja/powersync/latest/docs)
- [Getting Started guide](docs/guides/getting-started.md)
- [Connecting Supabase guide](docs/guides/connecting-supabase.md)
- [`powersync_instance` reference](docs/resources/instance.md)
- [`powersync_project` reference](docs/resources/project.md)

## Examples

Runnable examples live in [`examples/`](examples/). Each example is a self-contained directory you can `cd` into and `terraform apply`. They cover every resource and data source the provider exposes.

## Development

### Requirements

- Go 1.22+
- Terraform 1.5+

### Building locally

```sh
make build       # compile the provider
make install     # install the binary to $GOPATH/bin
make test        # run unit tests
make testacc     # run acceptance tests (requires staging credentials, hits the real API)
```

### Trying changes against a real Terraform config

The provider isn't loaded by name during development — instead, point Terraform at your locally-built binary using a [`dev_overrides`](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers) block.

A sample `dev.tfrc` lives in `playground/`. Use it like this:

```sh
make install     # rebuild the provider
cd playground
TF_CLI_CONFIG_FILE=dev.tfrc terraform plan
```

Terraform will print a warning about provider development overrides being in effect which is expected.

### Regenerating docs

```sh
make generate    # runs `tfplugindocs generate`
```

Docs come from three sources: schema `Description` fields in the Go code, the example HCL files in `examples/`, and the templates in `templates/`. Re-run `make generate` after changing any of those.

## Contributing

Issues and pull requests welcome. For non-trivial changes, please open an issue to discuss the approach before sending a PR.
