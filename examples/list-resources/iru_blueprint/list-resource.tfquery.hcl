# Every blueprint.
list "iru_blueprint" "all" {
  provider         = iru
  include_resource = true
}

# Only the blueprint named "Core".
list "iru_blueprint" "core" {
  provider = iru
  config {
    name = "Core"
  }
}
