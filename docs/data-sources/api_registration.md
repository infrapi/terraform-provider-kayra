---
title: kayra_api_registration
weight: 10
page_title: "kayra_api_registration Data Source - kayra"
subcategory: ""
description: |-
  An API registered in Kayra's platform and region.
---

# kayra_api_registration (Data Source)

An API registered in Kayra's platform and region.

## Example Usage

```terraform
data "kayra_api_registration" "compute" {
  name = "compute.infrapi.io"
}

output "compute_version" {
  value = data.kayra_api_registration.compute.version
}
```

## Schema

### Required

- `name` (String) The API's group, its full name, compute.infrapi.io.

### Read-Only

- `active` (Boolean) Whether it is open.
- `allowed_namespaces` (List of String) The namespaces that may subscribe to it and use it; every one when empty.
- `canary_namespaces` (List of String) The namespaces an ongoing canary release reaches, set by its sync action only.
- `created_at` (String) When it was created.
- `dependencies` (Attributes List) The registrations it depends on, in any platform and region. (see [below for nested schema](#nestedatt--dependencies))
- `description` (String) What it is.
- `downstream` (Boolean) Whether other APIs call it.
- `identity` (Attributes) How the API's app proves it is the API, system:api:<name>, for a token of its own: one of jwks_from, jwks and federated_credentials. (see [below for nested schema](#nestedatt--identity))
- `mode` (String) How the reconciler tells its app, a key of reconciler.mode in Kayra's configuration; reconciler.default_mode when empty.
- `owner` (String) Who owns it, a team.
- `platform` (String) Kayra's platform.
- `region` (String) Kayra's region.
- `short_name` (String) A one-label alias, compute, unique in the platform and region, for display only.
- `sync_id` (String) Its reconcile trigger: a new value, a release id say, tells every active subscription to reconcile. Kayra sets one when empty at creation.
- `uid` (String) Set at creation, kept by every write; the API's tokens carry it.
- `updated_at` (String) When it was last written.
- `upstream` (Boolean) Whether it calls other APIs.
- `version` (String) The version its app serves, v1 when empty; its contract is read from /apis/<name>/<version>/-/openapi.json.

<a id="nestedatt--dependencies"></a>
### Nested Schema for `dependencies`

### Read-Only

- `name` (String) Its API's group.
- `platform` (String) Its platform.
- `region` (String) Its region.

<a id="nestedatt--identity"></a>
### Nested Schema for `identity`

### Read-Only

- `federated_credentials` (Attributes List) Its platform's identity tokens trusted as the API. (see [below for nested schema](#nestedatt--identity--federated_credentials))
- `jwks` (Attributes) Public keys, as a JSON Web Key Set. (see [below for nested schema](#nestedatt--identity--jwks))
- `jwks_from` (String) spec, the keys the app serves at -/jwks.json, read with its contract. One of: spec.

<a id="nestedatt--identity--federated_credentials"></a>
### Nested Schema for `identity.federated_credentials`

### Read-Only

- `audience` (String) What its aud must hold, Kayra's URL when empty.
- `claims` (Map of String) Other claims it must carry, each exactly, a boolean as "true".
- `issuer` (String) The token's iss, one of Kayra's federation.issuers: https://gitlab.example.com.
- `name` (String) What tells it apart, gitlab-main.
- `subject` (String) Its sub, exactly.

<a id="nestedatt--identity--jwks"></a>
### Nested Schema for `identity.jwks`

### Read-Only

- `keys` (Attributes List) The keys. (see [below for nested schema](#nestedatt--identity--jwks--keys))

<a id="nestedatt--identity--jwks--keys"></a>
### Nested Schema for `identity.jwks.keys`

### Read-Only

- `alg` (String) Its algorithm, RS256.
- `crv` (String) An EC key's curve: P-256, P-384, P-521.
- `e` (String) An RSA key's exponent.
- `kid` (String) Its id.
- `kty` (String) Its type: RSA or EC.
- `n` (String) An RSA key's modulus.
- `use` (String) What it is for, sig.
- `x` (String) An EC key's x.
- `y` (String) An EC key's y.

