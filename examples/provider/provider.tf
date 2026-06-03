terraform {
  required_providers {
    powersync = {
      source  = "powersync-ja/powersync"
      version = "~> 0.1"
    }
  }
}

provider "powersync" {
  # admin_token is picked up from the PS_PAT_TOKEN environment variable (recommended).
  # It can also be inlined — less secure, but if inlined via a variable the value
  # itself can still be passed securely through Terraform's own env vars
  # (e.g. TF_VAR_admin_token).
  # admin_token = var.admin_token
}
