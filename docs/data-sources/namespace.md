---
page_title: "kayra_namespace Data Source - kayra"
subcategory: ""
description: |-
  A namespace.
---

# kayra_namespace (Data Source)

A namespace.

## Example Usage

```terraform
data "kayra_namespace" "core" {
  name = "core"
}
```

## Schema

### Required

- `name` (String) Its name, a DNS label.

### Read-Only

- `active` (Boolean) Set by the reconciler once the namespace is provisioned.
- `created_at` (String) When it was created.
- `updated_at` (String) When it was last written.

