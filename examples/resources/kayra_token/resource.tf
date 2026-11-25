# a personal access token of the provider's user, a person's: a personal
# access token creates no other
resource "kayra_token" "reports" {
  name       = "reports"
  expires_in = "720h"
  scope = {
    apis  = ["compute.infrapi.io"]
    verbs = ["get", "list"]
  }
}

output "reports_token" {
  value     = kayra_token.reports.token
  sensitive = true
}
