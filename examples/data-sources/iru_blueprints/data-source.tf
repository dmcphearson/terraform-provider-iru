data "iru_blueprints" "core" {
  name = "Core"
}

# Reference the matching ID:
#   one(data.iru_blueprints.core.results).id
