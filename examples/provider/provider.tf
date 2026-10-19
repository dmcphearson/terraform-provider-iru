terraform {
  required_providers {
    iru = {
      source = "dmcphearson/iru"
    }
  }
}

provider "iru" {
  # Credentials are read from the environment by default, for example from a
  # direnv .envrc that pulls the token from 1Password:
  #   export IRU_API_URL="https://acme.api.kandji.io"
  #   export IRU_API_TOKEN="$(op read 'op://Vault/Iru API/credential')"
}
