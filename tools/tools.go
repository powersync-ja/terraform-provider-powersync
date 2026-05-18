//go:build tools

// Package tools pins build-time tool dependencies so they are tracked in go.mod
// and `go generate` can resolve them. This file is never compiled into the provider
// binary (the `tools` build tag excludes it).
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
