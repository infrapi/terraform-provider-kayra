---
title: kayra_api_registration_status
weight: 10
page_title: "kayra_api_registration_status Data Source - kayra"
subcategory: ""
description: |-
  Where each active subscription to an API is, against its sync_id.
---

# kayra_api_registration_status (Data Source)

Where each active subscription to an API is, against its sync_id.

## Example Usage

```terraform
# where each active subscription to compute.infrapi.io is, against its sync_id
data "kayra_api_registration_status" "compute" {
  name = "compute.infrapi.io"
}

output "unsynced" {
  value = [for s in data.kayra_api_registration_status.compute.subscriptions : s.namespace if !s.synced]
}
```

## Schema

### Required

- `name` (String) The API's group.

### Read-Only

- `subscriptions` (Attributes List) Its active subscriptions, by namespace. (see [below for nested schema](#nestedatt--subscriptions))
- `sync_id` (String) Its current sync_id.

<a id="nestedatt--subscriptions"></a>
### Nested Schema for `subscriptions`

### Read-Only

- `current_sync_id` (String) The sync_id it was last reconciled to.
- `namespace` (String) Its namespace.
- `state` (String) Its last sync's outcome: init, success or error.
- `synced` (Boolean) Whether that is the API's.

