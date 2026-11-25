terraform {
  required_providers {
    kayra = {
      source = "infrapi/kayra"
    }
  }
}

# KAYRA_URL and KAYRA_TOKEN (or KAYRA_TOKEN_FILE) when left out.
provider "kayra" {
  url        = "https://kayra.example.com"
  token_file = pathexpand("~/.config/infrapi/kayra.token")
}
