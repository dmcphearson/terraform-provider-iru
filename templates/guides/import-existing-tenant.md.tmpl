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

Each list resource takes an optional filter: `name` for most types, `blueprint_id`
for `iru_blueprint_assignment` and `iru_device`, and `device_id` for
`iru_device_note`. `iru_blueprint_routing` needs none.

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
- **`iru_device`:** devices can only be imported, never created, and destroying one only
  removes it from state. Remove `tags`, `user_id` or `asset_tag` from the generated
  config if you don't want Terraform to manage them.
- Replace the IDs in references with resource references, for example
  `blueprint_id = iru_blueprint.core.id`, so Terraform orders changes correctly.

## 4. Plan and apply

```shell
terraform plan
```

The plan should show imports and few or no changes. Apply it, then remove the import
blocks.

## Coming from another Iru provider

State from another provider can't be handed over: resource names may match, but the
schemas differ. Re-import instead. Nothing in Iru changes along the way.

1. Back up state, then remove the old provider's resources from it. `state rm` only
   edits state; it never touches Iru.

   ```shell
   terraform state pull > before-migration.tfstate
   terraform state list | grep '^iru_' | xargs -n1 terraform state rm
   ```

2. Set the provider source to `dmcphearson/iru` in `required_providers` and run
   `terraform init -upgrade`.
3. Generate imports and config with steps 1 and 2 above. Either keep the generated
   config, or keep the import blocks and adapt your existing config.
4. Plan. It should show only imports; fix any attribute it wants to change.

Coming from `MScottBlake/iru`, these differ:

- `iru_blueprint_library_item` is `iru_blueprint_assignment`, with the same attributes.
- `enrollment_code_active` on `iru_blueprint` and `iru_blueprint_routing` is
  `enrollment_code = { is_active = ... }`; the code itself is `enrollment_code.code`.
- `iru_device`: `device_name` is `name`.
- `iru_in_house_app` uploads from `source` (a local `.ipa`) instead of `file_key`.
- `iru_custom_profile` has no `runs_on_vision`.
- ADE devices and integrations and Prism exports aren't supported yet. Exclude them
  from the `state rm` in step 1 and keep them on the old provider.
