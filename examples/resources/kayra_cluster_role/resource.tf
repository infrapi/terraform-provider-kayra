resource "kayra_cluster_role" "catalog_reader" {
  name = "catalog-reader"
  rules = [{
    apis      = ["catalog.infrapi.io"]
    resources = ["apiregistrations", "apiregistrations/status"]
    verbs     = ["get", "list"]
  }]
}
