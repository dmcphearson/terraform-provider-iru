data "iru_blueprint_routing_activity" "recent" {
  limit = 50
}

# Devices routed in the last 50 events, with the blueprint each landed on.
output "routed" {
  value = [
    for e in data.iru_blueprint_routing_activity.recent.results :
    { device = e.device_id, blueprint = jsondecode(e.details).destination_blueprint.name }
    if e.activity_type == "routing_departure"
  ]
}
