# terraform-provider-kayra

The [Kayra](https://infrapi.github.io/products/kayra/) provider for OpenTofu
and Terraform: namespaces, API registrations and subscriptions, the RBAC
(cluster and namespace groups, roles and role bindings), service accounts
and personal access tokens, as resources.

Each resource is one of Kayra's objects, and each of its attributes one
property of that object, as Kayra's contract (OpenAPI) describes it: a plan
shows a change per property, never a JSON blob. The schemas are generated
from Kayra's contracts, [internal/contracts](internal/contracts).

## Use it

On the [OpenTofu registry](https://search.opentofu.org/provider/infrapi/kayra)
and the [Terraform registry](https://registry.terraform.io/providers/infrapi/kayra):
the same `source`, each tool resolves it to its own registry.

```terraform
terraform {
  required_providers {
    kayra = {
      source = "infrapi/kayra"
    }
  }
}

provider "kayra" {
  url = "https://kayra.example.com" # or KAYRA_URL
  # token or token_file, or KAYRA_TOKEN / KAYRA_TOKEN_FILE
}

resource "kayra_namespace" "acme" {
  name = "acme"
}

resource "kayra_subscription" "acme_compute" {
  namespace = kayra_namespace.acme.name
  api       = "compute.infrapi.io"
}
```

The token is one of Kayra's: a personal access token
(`infrapi-cli create tokens -d '{"name": "tofu", "expires_in": "720h"}'`), or
a service account's, a CI job's say. The provider does what the token's user
may, as Kayra's RBAC decides. The groups' paths come from Kayra's
`/.well-known/kayra.json`, whatever its domain.

Every resource, data source and attribute: [docs/](docs/index.md), examples
in [examples/](examples).

## Develop

You need Go 1.27 and OpenTofu (or Terraform).

1. Build and install the provider in `~/go/bin`:

   ```bash
   make install
   ```

2. Point OpenTofu at it, in `~/.tofurc` (`~/.terraformrc` for Terraform):

   ```hcl
   provider_installation {
     dev_overrides {
       "infrapi/kayra" = "/home/<you>/go/bin"
     }
     direct {}
   }
   ```

   `tofu plan` and `tofu apply` then use it, with no `tofu init`.

3. After a change of Kayra's contracts, copy them and generate the schemas
   and the docs again:

   ```bash
   make contracts        # from ../kayra/api; KAYRA=<path> otherwise
   ```

   `make generate` alone generates `internal/provider/zz_generated.go` from
   `internal/contracts`, then `docs/` from the schemas and `examples/`. The
   CI refuses either out of date.

4. Run the checks: `make all` (generate, fmt, lint, security, test, build).

### Acceptance tests

They create, update, import and delete a namespace and its RBAC in a real
Kayra, a dev stack's say (`infrapi-cli dev up`), with a sysadmin's token:

```bash
export KAYRA_URL=http://localhost:9080 KAYRA_TOKEN=<a sysadmin's token>
make testacc
```

### Against a Kayra, without OpenTofu

`TestLive` runs every resource's operations, create, read, update, import
and delete, in process against a real Kayra, with a sysadmin's token: no
`tofu` needed. The namespace `acme` must exist.

```bash
KAYRA_LIVE_URL=http://localhost:9080 KAYRA_LIVE_TOKEN=<a sysadmin's token> \
  go test -count=1 -run TestLive ./internal/provider/
```

`KAYRA_LIVE_ISSUER`, one of the Kayra's `federation.issuers`, adds a
federated credential to the service account.

## Release

[RELEASING.md](RELEASING.md): a `feat:` or `fix:` merged to `main`, a
signed GitHub release, and the registries.
