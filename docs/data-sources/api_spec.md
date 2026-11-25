---
title: kayra_api_spec
weight: 10
page_title: "kayra_api_spec Data Source - kayra"
subcategory: ""
description: |-
  An API's specification, the OpenAPI document of what it serves, its contract and its operational endpoints: for an API registered in Kayra's catalog, the one Kayra's reconciler last read from it; for one of Kayra's own groups, the one Kayra serves. A gateway's routes, one per path, come from it.
---

# kayra_api_spec (Data Source)

An API's specification, the OpenAPI document of what it serves, its contract and its operational endpoints: for an API registered in Kayra's catalog, the one Kayra's reconciler last read from it; for one of Kayra's own groups, the one Kayra serves. A gateway's routes, one per path, come from it.

## Example Usage

```terraform
# What compute.infrapi.io serves, as Kayra's reconciler last read it: a
# gateway route per path, its methods only.
data "kayra_api_spec" "compute" {
  api = "compute.infrapi.io"
}

output "compute_routes" {
  value = data.kayra_api_spec.compute.available ? {
    for p in data.kayra_api_spec.compute.paths : p.path => p.methods
  } : {}
}

# One of Kayra's own groups: the specification Kayra serves.
data "kayra_api_spec" "authorization" {
  api = "authorization.infrapi.io"
}
```

## Schema

### Required

- `api` (String) The API's group, compute.infrapi.io, or one of Kayra's own (authorization.<domain>).

### Optional

- `version` (String) Its version, v1 when unset: one of Kayra's own groups only; a registration's is its own.

### Read-Only

- `available` (Boolean) Whether there is a specification: false for an API not registered yet, or Kayra's reconciler has not read yet, its paths empty. A configuration registering the API and routing from its specification gets its routes at the apply after Kayra read it.
- `openapi` (String) The specification, an OpenAPI document as JSON; empty when not available.
- `paths` (Attributes List) Its paths, sorted, each with the methods it serves. (see [below for nested schema](#nestedatt--paths))
- `registered` (Boolean) Whether the API is registered in Kayra's catalog: false for one of Kayra's own groups, and for an API not registered yet.
- `root` (String) Its paths' root, /apis/<api>/<version>.

<a id="nestedatt--paths"></a>
### Nested Schema for `paths`

### Read-Only

- `methods` (List of String) Its methods, upper case, sorted: GET, POST.
- `path` (String) The path, as OpenAPI writes it: /apis/compute.infrapi.io/v1/namespaces/{namespace}/vms/{name}.

