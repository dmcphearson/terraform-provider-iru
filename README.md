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
| `iru_custom_app` | resource |
| `iru_blueprint` | resource |
| `iru_blueprint_assignment` | resource |
| `iru_in_house_app` | resource |
| `iru_self_service_categories` | data source |


## Development

```bash
make build      # compile
make test       # unit tests (mocked, no token needed)
make install    # install to $GOPATH/bin for dev_overrides
```

To run against a local build, add a `dev_overrides` block to `~/.terraformrc`
pointing `dmcphearson/iru` at your `$GOPATH/bin`, then run Terraform normally
(skip `terraform init`).

## Offline and MDM-managed installation (filesystem mirror)

For Macs that can't reach the public registry, or where you want the provider
version pinned centrally, you can install it into Terraform's filesystem mirror
with Iru itself. `make mirror-zip` builds a zip containing the macOS `arm64` +
`amd64` binaries in Terraform's unpacked filesystem-mirror layout:

```
registry.terraform.io/dmcphearson/iru/<version>/darwin_arm64/terraform-provider-iru_v<version>
registry.terraform.io/dmcphearson/iru/<version>/darwin_amd64/terraform-provider-iru_v<version>
```

Deploy it as an Iru custom app that unzips into the machine-wide mirror
`/Library/Application Support/io.terraform/plugins`, which every user's Terraform
searches automatically. Example custom app scripts are in
[`examples/mirror-install`](./examples/mirror-install): `preinstall.sh` installs a
pinned, checksum-verified Terraform if none is present, and `postinstall.sh` strips
Gatekeeper quarantine, fixes permissions and ad-hoc signs the binaries if needed.
`sudo scripts/simulate-iru-install.sh` runs the same install locally to test them. `terraform init` then resolves `dmcphearson/iru` from the
local mirror with a real lockfile and version pin, with no registry access and no
`dev_overrides`.

Shipping a new version:

```bash
make mirror-zip VERSION=0.1.0        # produces dist/iru-provider-0.1.0-mirror.zip
```

1. Upload the zip to Iru (console, or the `upload-custom-app` API).
2. Copy the returned `file_key`.
3. Set it as the custom app's `file_key` in your Iru Terraform configuration and
   `terraform apply`. Devices pick up the new binary from Self Service.

## License

[MPL-2.0](./LICENSE).

## Acknowledgements

Built independently against the public Iru/Kandji REST API and the HashiCorp
Terraform Plugin Framework. Iru and Kandji are trademarks of their respective owners;
this provider is not affiliated with or endorsed by Kandji, Inc.
