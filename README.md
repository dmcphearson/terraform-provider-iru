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

## Internal distribution (filesystem mirror via Iru)

This provider is distributed to Fluency devices internally rather than through the
public Terraform Registry. `make mirror-zip` builds a zip containing the macOS
`arm64` + `amd64` binaries in Terraform's unpacked filesystem-mirror layout:

```
registry.terraform.io/dmcphearson/iru/<version>/darwin_arm64/terraform-provider-iru_v<version>
registry.terraform.io/dmcphearson/iru/<version>/darwin_amd64/terraform-provider-iru_v<version>
```

An Iru custom app unzips this into the machine-wide mirror
`/Library/Application Support/io.terraform/plugins`, which every user's Terraform
searches automatically. After install, `terraform init` resolves `dmcphearson/iru`
from the local mirror with a real lockfile and version pin — no registry, no
`dev_overrides`.

Shipping a new version:

```bash
make mirror-zip VERSION=0.1.0        # produces dist/iru-provider-0.1.0-mirror.zip
```

1. Upload the zip to Iru (console, or the `upload-custom-app` API).
2. Copy the returned `file_key`.
3. Set it as `iru_provider_file_key` in the `fluency-iru-config` `apps` component and
   `terraform apply`. Devices pick up the new binary from Self Service.

## License

[MPL-2.0](./LICENSE).

## Acknowledgements

Built independently against the public Iru/Kandji REST API and the HashiCorp
Terraform Plugin Framework. Iru and Kandji are trademarks of their respective owners;
this provider is not affiliated with or endorsed by Kandji, Inc.
