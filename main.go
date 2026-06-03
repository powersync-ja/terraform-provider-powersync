package main

// Generate Terraform Registry documentation from schema descriptions + the examples/ directory.
// Note: the next line is a Go compiler directive — the colon must immediately follow `//go` (no space),
// or the directive is treated as a regular comment and `go generate` becomes a no-op.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name powersync

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/powersync/terraform-provider-powersync/internal/provider"
)

var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/powersync-ja/powersync",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err)
	}
}
