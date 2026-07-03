terraform {
  required_providers {
    iru = {
      source = "dmcphearson/iru"
    }
  }
}

provider "iru" {
  # Credentials are read from the environment by default:
  #   export IRU_API_URL="https://acme.api.kandji.io"
  #   export IRU_API_TOKEN="..."
}
