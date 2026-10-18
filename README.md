# Terraform Provider for Iru (Kandji)

A Terraform provider for [Iru](https://www.iru.com) (formerly Kandji) endpoint
management: blueprints and their assignments, custom scripts, custom profiles, custom
and in-house apps, and tags, as code.

Published to the Terraform Registry as
[`dmcphearson/iru`](https://registry.terraform.io/providers/dmcphearson/iru), with
full documentation there.

- **Import works on every resource**, by ID or by resource identity, so existing
  tenants can be brought under Terraform.
- **Apps upload from a local file.** Point `source` at an installer or `.ipa`; the
  provider uploads it and re-uploads only when the file's SHA256 changes.
- **Known API rules fail at plan, not halfway through apply.** Settings the Iru API
  is known to reject are caught during `terraform plan` with a clear message.

## Requirements

- Terraform 1.0 or later. Import by resource identity needs Terraform 1.12 or later.
- An Iru API token with permissions for the endpoints you manage.

OpenTofu is not verified yet.

## Usage

```hcl
terraform {
  required_providers {
    iru = {
      source  = "dmcphearson/iru"
      version = "~> 0.2"
    }
  }
}

provider "iru" {
  # Read from the environment by default:
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

The provider talks to one Iru tenant with a bearer API token. Set `IRU_API_URL` and
`IRU_API_TOKEN` in the environment (preferred), or the `api_url` and `api_token`
provider arguments. Grant the token API permissions for the objects you manage, for
example library items, blueprints and tags.

## Resources and data sources

| Name | Kind | Manages |
|------|------|---------|
| [`iru_blueprint`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/resources/blueprint) | resource | Blueprints (new ones are `map`; existing classic blueprints can be imported) |
| [`iru_blueprint_assignment`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/resources/blueprint_assignment) | resource | Assign a library item to a blueprint |
| [`iru_custom_app`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/resources/custom_app) | resource | Custom apps; uploads the installer from a local `source` path |
| [`iru_in_house_app`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/resources/in_house_app) | resource | In-house `.ipa` apps for iPhone, iPad and Apple TV; uploads from `source` |
| [`iru_custom_profile`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/resources/custom_profile) | resource | Custom configuration profiles from a `.mobileconfig` |
| [`iru_custom_script`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/resources/custom_script) | resource | Custom scripts, with optional remediation |
| [`iru_tag`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/resources/tag) | resource | Tags |
| [`iru_blueprints`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/data-sources/blueprints) | data source | List blueprints, optionally by name |
| [`iru_custom_profiles`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/data-sources/custom_profiles) | data source | List custom profiles, optionally by name |
| [`iru_custom_scripts`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/data-sources/custom_scripts) | data source | List custom scripts, optionally by name |
| [`iru_tags`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/data-sources/tags) | data source | List tags, optionally by name |
| [`iru_self_service_categories`](https://registry.terraform.io/providers/dmcphearson/iru/latest/docs/data-sources/self_service_categories) | data source | Look up Self Service category IDs |

## Importing existing objects

Every resource imports by its Iru ID, except assignments (below). With Terraform 1.12 or later you can also use
resource identity:

```hcl
import {
  to       = iru_tag.engineering
  identity = { id = "00000000-0000-0000-0000-000000000000" }
}
```

Blueprint assignments import as `blueprint_id:library_item_id`, plus
`:assignment_node_id` on map blueprints that use conditional logic.

## Known API limits

The Iru API enforces these. The provider checks the first three at plan time.

- New blueprints must be `type = "map"`. Classic blueprints can be imported and
  managed but not created.
- Scripts and custom apps with `no_enforcement` must set `show_in_self_service = true`.
- `self_service_recommended` requires `show_in_self_service = true`.
- Only active library items can be assigned to a blueprint.

## Status

Actively maintained. Coverage grows each release; open an issue to request a resource
or report a bug.

## Installing on managed Macs

To deploy the provider through Iru itself, as a filesystem mirror with no registry
access, see the
[filesystem mirror guide](./docs/guides/filesystem-mirror.md).

## Development

```bash
make build      # compile
make test       # unit tests (mocked, no token needed)
make install    # install to $GOPATH/bin for dev_overrides
make generate   # regenerate docs/ from schemas, examples and templates/
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
