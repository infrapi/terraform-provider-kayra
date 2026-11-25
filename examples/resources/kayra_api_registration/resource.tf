resource "kayra_api_registration" "compute" {
  name        = "compute.infrapi.io"
  short_name  = "compute"
  owner       = "team-compute"
  description = "Virtual machines, hosts and flavors."
  active      = true

  # the app serves its public keys at -/jwks.json
  identity = {
    jwks_from = "spec"
  }

  # a new value tells every active subscription to reconcile
  sync_id = "release-2026-11-25"
}
