# Attach a library item to a blueprint. For map blueprints with conditional
# logic, also set assignment_node_id.
resource "iru_blueprint_assignment" "core_wifi" {
  blueprint_id    = iru_blueprint.core.id
  library_item_id = iru_custom_profile.wifi.id
}
