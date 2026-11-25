# where each active subscription to compute.infrapi.io is, against its sync_id
data "kayra_api_registration_status" "compute" {
  name = "compute.infrapi.io"
}

output "unsynced" {
  value = [for s in data.kayra_api_registration_status.compute.subscriptions : s.namespace if !s.synced]
}
