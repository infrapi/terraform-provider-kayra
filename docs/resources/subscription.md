---
page_title: "kayra_subscription Resource - kayra"
subcategory: ""
description: |-
  A namespace's subscription to an API.
---

# kayra_subscription (Resource)

A namespace's subscription to an API.

## Example Usage

```terraform
resource "kayra_subscription" "acme_compute" {
  namespace = kayra_namespace.acme.name
  api       = kayra_api_registration.compute.name
}
```

## Schema

### Required

- `api` (String) The API's group, compute.infrapi.io; never its short name.
- `namespace` (String) The namespace subscribing.

### Read-Only

- `active` (Boolean) Set once, by its first successful sync; until then the namespace's requests to the API are refused.
- `created_at` (String) When it was created.
- `current_sync_id` (String) The API's sync_id it was last reconciled to.
- `expected_sync_id` (String) The API's current sync_id.
- `id` (String) Its id, its import id: its keys joined by /.
- `state` (String) Its last sync's outcome: init until the first one. One of: init, success, error.
- `synced` (Boolean) Whether it is active and at the API's sync_id.
- `updated_at` (String) When it was last written.

## Import

Import is supported using the following syntax:

```shell
# a subscription by its namespace and API
tofu import kayra_subscription.acme_compute acme/compute.infrapi.io
```
