resource "kayra_subscription" "acme_compute" {
  namespace = kayra_namespace.acme.name
  api       = kayra_api_registration.compute.name
}
