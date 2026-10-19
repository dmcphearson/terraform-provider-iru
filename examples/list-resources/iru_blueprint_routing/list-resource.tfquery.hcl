# The tenant's routing setting, with config for import.
list "iru_blueprint_routing" "this" {
  provider         = iru
  include_resource = true
}
