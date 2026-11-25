---
page_title: "Provider: Kayra"
description: |-
  Manage Kayra: namespaces, API registrations and subscriptions, the RBAC, service accounts and personal access tokens.
---

# Kayra Provider

Manage Kayra: namespaces, API registrations and subscriptions, the RBAC, service accounts and personal access tokens.

Kayra is InfraPI's identity, namespaces, RBAC and API catalog: <https://infrapi.github.io/products/kayra/>. Every attribute of a resource is a property of Kayra's object, as its contract (OpenAPI) describes it, so a plan shows a change per property.

The provider calls Kayra with a token of Kayra's: a personal access token (`infrapi-cli create tokens`), or a service account's, a CI job's say. Its rights are its user's, decided by Kayra's RBAC.

## Example Usage

```terraform
terraform {
  required_providers {
    kayra = {
      source = "infrapi/kayra"
    }
  }
}

# KAYRA_URL and KAYRA_TOKEN (or KAYRA_TOKEN_FILE) when left out.
provider "kayra" {
  url        = "https://kayra.example.com"
  token_file = pathexpand("~/.config/infrapi/kayra.token")
}
```

## Schema

### Optional

- `token` (String, Sensitive) A token of Kayra's: a personal access token, or a service account's; `KAYRA_TOKEN` when unset.
- `token_file` (String) A file holding the token, read at each run; `KAYRA_TOKEN_FILE` when unset. `token` wins.
- `url` (String) Kayra's URL, `https://kayra.example.com`; `KAYRA_URL` when unset. Its groups are discovered at `/.well-known/kayra.json`.

