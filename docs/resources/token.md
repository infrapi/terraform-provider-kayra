---
page_title: "kayra_token Resource - kayra"
subcategory: ""
description: |-
  A user's personal access token.
---

# kayra_token (Resource)

A user's personal access token.

## Example Usage

```terraform
# a personal access token of the provider's user, a person's: a personal
# access token creates no other
resource "kayra_token" "reports" {
  name       = "reports"
  expires_in = "720h"
  scope = {
    apis  = ["compute.infrapi.io"]
    verbs = ["get", "list"]
  }
}

output "reports_token" {
  value     = kayra_token.reports.token
  sensitive = true
}
```

## Schema

### Required

- `name` (String) What it is for, laptop: 64 characters at most.

### Optional

- `expires_in` (String) Its lifetime, a duration: 720h; Kayra's default when empty.
- `scope` (Attributes) What a personal access token may do, whatever its user's roles; each entry may be "*". (see [below for nested schema](#nestedatt--scope))

### Read-Only

- `created_at` (String) When it was created.
- `expires_at` (String) When it expires.
- `id` (String) Its id.
- `last_used_at` (String) When it was last used.
- `token` (String, Sensitive) Its secret, at its creation only.
- `user` (String) Its user.

<a id="nestedatt--scope"></a>
### Nested Schema for `scope`

### Required

- `apis` (List of String) API groups, compute.infrapi.io.
- `verbs` (List of String) get, list, create, update, patch, delete.

## Import

Import is supported using the following syntax:

```shell
# a token by its id; its secret is not read back
tofu import kayra_token.reports <id>
```
