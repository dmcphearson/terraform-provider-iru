---
page_title: "Install through Iru (filesystem mirror)"
subcategory: ""
description: |-
  Deploy the provider to managed Macs as an Iru custom app, without registry access.
---

# Install through Iru (filesystem mirror)

For Macs that can't reach the public registry, or where you want the provider
version pinned centrally, you can install it into Terraform's filesystem mirror
with Iru itself. From a clone of this repo, `make mirror-zip` builds a zip containing the macOS `arm64` +
`amd64` binaries in Terraform's unpacked filesystem-mirror layout:

```
registry.terraform.io/dmcphearson/iru/<version>/darwin_arm64/terraform-provider-iru_v<version>
registry.terraform.io/dmcphearson/iru/<version>/darwin_amd64/terraform-provider-iru_v<version>
```

Deploy it as an Iru custom app that unzips into the machine-wide mirror
`/Library/Application Support/io.terraform/plugins`, which every user's Terraform
searches automatically. Example custom app scripts are in
[`examples/mirror-install`](https://github.com/dmcphearson/terraform-provider-iru/tree/main/examples/mirror-install): `preinstall.sh` installs a
pinned, checksum-verified Terraform if none is present, and `postinstall.sh` strips
Gatekeeper quarantine, fixes permissions and ad-hoc signs the binaries if needed.
`sudo scripts/simulate-iru-install.sh` (in the repo) runs the same install locally to test them.
`terraform init` then resolves `dmcphearson/iru` from the
local mirror with a real lockfile and version pin, with no registry access and no
`dev_overrides`.

When the CLI config has no `provider_installation` block, Terraform uses a provider
found in that mirror exclusively, so versions that aren't in the mirror, such as a
pre-release, report "no available releases match". To pull one from the registry,
point `TF_CLI_CONFIG_FILE` at a config that installs directly:

```hcl
provider_installation {
  direct {}
}
```

Shipping a new version:

```bash
make mirror-zip VERSION=0.2.0        # produces dist/iru-provider-0.2.0-mirror.zip
```

Point the custom app's `source` at the zip in your Iru Terraform configuration and
`terraform apply`. The provider uploads it, and devices pick up the new binary from
Self Service.
