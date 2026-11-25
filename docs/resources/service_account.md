---
page_title: "kayra_service_account Resource - kayra"
subcategory: ""
description: |-
  A namespace's machine identity, system:serviceaccount:<namespace>:<name>. It holds no secret: its tokens are asked for, short-lived.
---

# kayra_service_account (Resource)

A namespace's machine identity, system:serviceaccount:<namespace>:<name>. It holds no secret: its tokens are asked for, short-lived.

## Example Usage

```terraform
# acme's deployment pipeline: a GitHub Actions job of acme/compute's main
# branch gets its tokens, with no secret
resource "kayra_service_account" "deployer" {
  namespace   = kayra_namespace.acme.name
  name        = "deployer"
  description = "acme/compute's deployments."
  federated_credentials = [{
    name    = "github-main"
    issuer  = "https://token.actions.githubusercontent.com"
    subject = "repo:acme/compute:ref:refs/heads/main"
    claims = {
      repository = "acme/compute"
    }
  }]
}
```

## Schema

### Required

- `name` (String) Its name, a DNS label.
- `namespace` (String) The namespace it lives in.

### Optional

- `description` (String) What it is.
- `federated_credentials` (Attributes List) Platforms' identity tokens trusted as it, a CI job's say. (see [below for nested schema](#nestedatt--federated_credentials))
- `jwks` (Attributes) Public keys, as a JSON Web Key Set. (see [below for nested schema](#nestedatt--jwks))
- `metadata` (Attributes) Where an RBAC object applies. (see [below for nested schema](#nestedatt--metadata))

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) Its id, its import id: its keys joined by /.
- `uid` (String) Set at creation; its tokens carry it, so a deleted one's are refused.
- `updated_at` (String) When it was last written.

<a id="nestedatt--federated_credentials"></a>
### Nested Schema for `federated_credentials`

### Required

- `issuer` (String) The token's iss, one of Kayra's federation.issuers: https://gitlab.example.com.
- `name` (String) What tells it apart, gitlab-main.

### Optional

- `audience` (String) What its aud must hold, Kayra's URL when empty.
- `claims` (Map of String) Other claims it must carry, each exactly, a boolean as "true".
- `subject` (String) Its sub, exactly.

<a id="nestedatt--jwks"></a>
### Nested Schema for `jwks`

### Required

- `keys` (Attributes List) The keys. (see [below for nested schema](#nestedatt--jwks--keys))

<a id="nestedatt--jwks--keys"></a>
### Nested Schema for `jwks.keys`

### Required

- `kty` (String) Its type: RSA or EC.

### Optional

- `alg` (String) Its algorithm, RS256.
- `crv` (String) An EC key's curve: P-256, P-384, P-521.
- `e` (String) An RSA key's exponent.
- `kid` (String) Its id.
- `n` (String) An RSA key's modulus.
- `use` (String) What it is for, sig.
- `x` (String) An EC key's x.
- `y` (String) An EC key's y.

<a id="nestedatt--metadata"></a>
### Nested Schema for `metadata`

### Optional

- `namespace` (String) Its namespace. A namespace object's is the path's when empty. A cluster object is global when empty, scoped to the namespace otherwise.
- `platform` (String) Its platform.
- `region` (String) Its region.

## Import

Import is supported using the following syntax:

```shell
tofu import kayra_service_account.deployer acme/deployer
```
