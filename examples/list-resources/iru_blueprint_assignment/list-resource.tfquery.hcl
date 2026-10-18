# Every library item assignment, on every blueprint.
list "iru_blueprint_assignment" "all" {
  provider         = iru
  include_resource = true
}

# Only the assignments on one blueprint.
list "iru_blueprint_assignment" "core" {
  provider = iru
  config {
    blueprint_id = "00000000-0000-0000-0000-000000000000"
  }
}
