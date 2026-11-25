// Command terraform-provider-kayra is the Kayra provider for OpenTofu and
// Terraform: go install github.com/infrapi/terraform-provider-kayra@latest.
// The same command as cmd/terraform-provider-kayra, which the CI and the
// release build.
package main

import (
	"log"

	"github.com/infrapi/terraform-provider-kayra/internal/provider"
)

// version is the release's, set by GoReleaser.
var version = "dev"

func main() {
	if err := provider.Main(version); err != nil {
		log.Fatal(err)
	}
}
