---
page_title: "kayra_cluster_group Resource - kayra"
subcategory: ""
description: |-
  Who: external groups, users, the namespace's service accounts and registered APIs. Any one list is enough to be a member; an entry of groups or users may be "*".
---

# kayra_cluster_group (Resource)

Who: external groups, users, the namespace's service accounts and registered APIs. Any one list is enough to be a member; an entry of groups or users may be "*".

## Example Usage

```terraform
# the platform team, on every cluster-scoped resource
resource "kayra_cluster_group" "platform" {
  name   = "platform"
  groups = ["platform-admins"]
}

# acme's admins: may manage acme's RBAC
resource "kayra_cluster_group" "acme_admins" {
  name  = "acme-admins"
  users = ["jdoe@example.com"]
  metadata = {
    namespace = kayra_namespace.acme.name
  }
}
```

## Schema

### Required

- `name` (String) Its name.

### Optional

- `apis` (List of String) Registered APIs, by group, calling with their own identity; exact names.
- `description` (String) What it is.
- `groups` (List of String) Groups, as Kayra resolves a caller's, sandbox-sre-sysadmin; never a DN.
- `metadata` (Attributes) Where an RBAC object applies. (see [below for nested schema](#nestedatt--metadata))
- `serviceaccounts` (List of String) Service accounts of the group's own namespace, by name; a namespace group's only.
- `users` (List of String) Users, by their id.

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
tofu import kayra_cluster_group.platform platform
```
