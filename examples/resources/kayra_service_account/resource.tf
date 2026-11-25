# acme's deployment pipeline: a GitHub Actions job of acme/compute's main
# branch gets its tokens, with no secret
resource "kayra_service_account" "deployer" {
  namespace   = kayra_namespace.acme.name
  name        = "deployer"
  description = "acme/compute's deployments."
  federated_credentials = [{
    name    = "github-main"
    issuer  = "https://token.actions.githubusercontent.com"
    subject = "repo:acme/compute:ref:refs/heads/main"
    claims = {
      repository = "acme/compute"
    }
  }]
}
