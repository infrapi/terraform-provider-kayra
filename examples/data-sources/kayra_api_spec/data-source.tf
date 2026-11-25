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
