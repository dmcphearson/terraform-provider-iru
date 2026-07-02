# Terraform Provider for Iru (Kandji)

A Terraform provider for [Iru](https://kandji.io) (formerly Kandji) endpoint
management. Manage library items — custom scripts, custom profiles, tags, custom
apps, and blueprints — as code.

Published to the Terraform Registry as
[`dmcphearson/iru`](https://registry.terraform.io/providers/dmcphearson/iru).

## Usage

```hcl
terraform {
  required_providers {
    iru = {
      source = "dmcphearson/iru"
    }
  }
}

provider "iru" {
  # api_url and api_token are read from the environment by default:
  #   export IRU_API_URL="https://acme.api.kandji.io"
  #   export IRU_API_TOKEN="..."   # keep tokens out of config
}

resource "iru_custom_script" "hello" {
  name                = "Hello"
  execution_frequency = "once"
  script              = file("${path.module}/hello.sh")
  active              = true
}
```

## Authentication

The provider authenticates to a single Iru tenant with a bearer API token. Provide
credentials via the `IRU_API_URL` / `IRU_API_TOKEN` environment variables (preferred)
or the `api_url` / `api_token` provider arguments. The token needs scopes for the
library item types you manage.

## Resources & data sources

| Type | Kind |
|------|------|
| `iru_custom_script` | resource |
| `iru_custom_profile` | resource |
| `iru_tag` | resource |
| `iru_self_service_categories` | data source |

Custom apps, IPA apps, and blueprints are in progress.

## Development

```bash
make build      # compile
make test       # unit tests (mocked, no token needed)
make install    # install to $GOPATH/bin for dev_overrides
```

To run against a local build, add a `dev_overrides` block to `~/.terraformrc`
pointing `dmcphearson/iru` at your `$GOPATH/bin`, then run Terraform normally
(skip `terraform init`).

## License

[MPL-2.0](./LICENSE).

## Acknowledgements

Built independently against the public Iru/Kandji REST API and the HashiCorp
Terraform Plugin Framework. Iru and Kandji are trademarks of their respective owners;
this provider is not affiliated with or endorsed by Kandji, Inc.
