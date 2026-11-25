package provider

import (
	"context"
	"flag"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Address is the provider's in the OpenTofu registry: what a debugger
// attaches to (-debug).
const Address = "registry.opentofu.org/infrapi/kayra"

// Main serves the provider to OpenTofu or Terraform, at version: the
// command's, terraform-provider-kayra, wherever it is built from (the
// module's root or cmd/terraform-provider-kayra).
func Main(version string) error {
	debug := flag.Bool("debug", false, "run with support for debuggers like delve")
	flag.Parse()
	return providerserver.Serve(context.Background(), New(version), providerserver.ServeOpts{
		Address: Address,
		Debug:   *debug,
	})
}
