// Command terraform-provider-kayra is the Kayra provider for OpenTofu and
// Terraform.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/infrapi/terraform-provider-kayra/internal/provider"
)

// version is the release's, set by GoReleaser.
var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "run with support for debuggers like delve")
	flag.Parse()
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.opentofu.org/infrapi/kayra",
		Debug:   *debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
