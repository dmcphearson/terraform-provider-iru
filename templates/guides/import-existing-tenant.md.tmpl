---
page_title: "Import an existing tenant"
subcategory: ""
description: |-
  Bring existing Iru objects under Terraform in bulk with terraform query.
---

# Import an existing tenant

Every resource in this provider has a list resource of the same name. With
Terraform 1.14 or later, `terraform query` uses them to find existing objects and
write import blocks and resource config for them, so a tenant that was set up by
hand can be adopted without writing an import block per object.

## 1. Write a query file

Create `tenant.tfquery.hcl` next to your configuration. Include the types you want
to adopt; `include_resource = true` is needed to generate config.

```terraform
list "iru_blueprint" "all" {
  provider         = iru
  include_resource = true
}

list "iru_custom_script" "all" {
  provider         = iru
  include_resource = true
}

list "iru_blueprint_assignment" "all" {
  provider         = iru
  include_resource = true
}
```

Each list resource takes an optional filter: `name` for most types, and
`blueprint_id` for `iru_blueprint_assignment`.

## 2. Generate config

```shell
terraform init
terraform query -generate-config-out=generated.tf
```

## 3. Review the generated config

Generated config is a starting point. Before planning:

- **`iru_in_house_app`:** set `source` to the path of the `.ipa`; it can't be read back
  from Iru. If the file matches what Iru holds, nothing is uploaded.
- **`iru_custom_app`:** the config references the uploaded binary by `file_key`. To
  manage the installer from a local file instead, replace `file_key` with `source`.
- **`iru_custom_profile`:** `profile_file` holds the profile inline. Consider moving
  it to a `.mobileconfig` file and using `file()`.
- **`iru_custom_script`:** `self_service_category_id` and `self_service_recommended`
  are write-only in the API, so set them by hand if you use them.
- **`iru_blueprint_assignment`:** node IDs on map blueprints can't be read back, so
  generated assignments have no `assignment_node_id`. Moving an item between nodes in
  the Iru UI doesn't cause drift.
- Replace the IDs in references with resource references, for example
  `blueprint_id = iru_blueprint.core.id`, so Terraform orders changes correctly.

## 4. Plan and apply

```shell
terraform plan
```

The plan should show imports and few or no changes. Apply it, then remove the import
blocks.
