resource "kayra_role_binding" "operators" {
  namespace = kayra_namespace.acme.name
  name      = "operators"
  role      = kayra_role.vm_operator.name
  groups    = [kayra_group.operators.name]
}
