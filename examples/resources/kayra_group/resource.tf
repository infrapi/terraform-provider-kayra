resource "kayra_group" "operators" {
  namespace       = kayra_namespace.acme.name
  name            = "operators"
  description     = "Who runs acme's virtual machines."
  groups          = ["acme-operators"]
  users           = ["jdoe@example.com"]
  serviceaccounts = [kayra_service_account.deployer.name]
}
