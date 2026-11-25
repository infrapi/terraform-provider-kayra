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
