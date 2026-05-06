terraform {
  required_providers {
    powersync = {
      source  = "powersync/powersync"
      version = "~> 0.1.0"
    }
  }
}

provider "powersync" {
  # admin_token is picked up from the PS_ADMIN_TOKEN environment variable (recommended).
  # It can also be inlined — less secure, but if inlined via a variable the value
  # itself can still be passed securely through Terraform's own env vars
  # (e.g. TF_VAR_ps_admin_token).
  # admin_token = var.ps_admin_token

  # Override for staging:
  # accounts_url = "https://staging-accounts.powersync.com"
}
