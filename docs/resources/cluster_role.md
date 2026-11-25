---
title: kayra_cluster_role
weight: 10
page_title: "kayra_cluster_role Resource - kayra"
subcategory: ""
description: |-
  A named set of rules.
---

# kayra_cluster_role (Resource)

A named set of rules.

## Example Usage

```terraform
resource "kayra_cluster_role" "catalog_reader" {
  name = "catalog-reader"
  rules = [{
    apis      = ["catalog.infrapi.io"]
    resources = ["apiregistrations", "apiregistrations/status"]
    verbs     = ["get", "list"]
  }]
}
```

## Schema

### Required

- `name` (String) Its name.

### Optional

- `description` (String) What it is.
- `metadata` (Attributes) Where an RBAC object applies. (see [below for nested schema](#nestedatt--metadata))
- `rules` (Attributes List) What it grants. (see [below for nested schema](#nestedatt--rules))

### Read-Only

- `created_at` (String) When it was created.
- `id` (String) Its id, its import id: its keys joined by /.
- `updated_at` (String) When it was last written.

<a id="nestedatt--metadata"></a>
### Nested Schema for `metadata`

### Optional

- `namespace` (String) Its namespace. A namespace object's is the path's when empty. A cluster object is global when empty, scoped to the namespace otherwise.
- `platform` (String) Its platform.
- `region` (String) Its region.

<a id="nestedatt--rules"></a>
### Nested Schema for `rules`

### Optional

- `apis` (List of String) API groups, compute.infrapi.io.
- `resources` (List of String) Resources, vms, or actions, vms/reboot.
- `verbs` (List of String) get, list, create, update, patch, delete; escalate and bind on roles and bindings.

## Import

Import is supported using the following syntax:

```shell
tofu import kayra_cluster_role.catalog_reader catalog-reader
```
