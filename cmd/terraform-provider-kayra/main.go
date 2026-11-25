// Command terraform-provider-kayra is the Kayra provider for OpenTofu and
// Terraform, as the CI and the release build it; the module's root main.go
// is the same command, for go install.
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
