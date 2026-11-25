resource "kayra_role" "vm_operator" {
  namespace = kayra_namespace.acme.name
  name      = "vm-operator"
  rules = [{
    apis      = ["compute.infrapi.io"]
    resources = ["vms", "vms/reboot"]
    verbs     = ["get", "list", "create", "update"]
  }]
}
