---
title: kayra_role_binding
weight: 10
page_title: "kayra_role_binding Resource - kayra"
subcategory: ""
description: |-
  A role granted to groups, of the same metadata.
---

# kayra_role_binding (Resource)

A role granted to groups, of the same metadata.

## Example Usage

```terraform
resource "kayra_role_binding" "operators" {
  namespace = kayra_namespace.acme.name
  name      = "operators"
  role      = kayra_role.vm_operator.name
  groups    = [kayra_group.operators.name]
}
```

## Schema

### Required

- `name` (String) Its name.
- `namespace` (String) The namespace it lives in.
- `role` (String) The role, by name.

### Optional

- `description` (String) What it is.
- `groups` (List of String) The groups, by name.
- `metadata` (Attributes) Where an RBAC object applies. (see [below for nested schema](#nestedatt--metadata))

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

## Import

Import is supported using the following syntax:

```shell
tofu import kayra_role_binding.operators acme/operators
```
