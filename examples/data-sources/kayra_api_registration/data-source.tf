data "kayra_api_registration" "compute" {
  name = "compute.infrapi.io"
}

output "compute_version" {
  value = data.kayra_api_registration.compute.version
}
