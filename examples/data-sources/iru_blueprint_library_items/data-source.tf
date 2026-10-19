# Everything assigned to a blueprint, including items assigned outside Terraform.
data "iru_blueprint_library_items" "core" {
  blueprint_id = iru_blueprint.core.id
}

output "core_items" {
  value = [for i in data.iru_blueprint_library_items.core.results : i.name]
}
