---
page_title: "kayra_namespace Resource - kayra"
subcategory: ""
description: |-
  A namespace.
---

# kayra_namespace (Resource)

A namespace.

## Example Usage

```terraform
resource "kayra_namespace" "acme" {
  name = "acme"
}
```

## Schema

### Required

- `name` (String) Its name, a DNS label.

### Read-Only

- `active` (Boolean) Set by the reconciler once the namespace is provisioned.
- `created_at` (String) When it was created.
- `id` (String) Its id, its import id: its keys joined by /.
- `updated_at` (String) When it was last written.

## Import

Import is supported using the following syntax:

```shell
# a namespace by its name
tofu import kayra_namespace.acme acme
```
