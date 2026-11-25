resource "kayra_cluster_role_binding" "platform_catalog" {
  name   = "platform-catalog"
  role   = kayra_cluster_role.catalog_reader.name
  groups = [kayra_cluster_group.platform.name]
}
