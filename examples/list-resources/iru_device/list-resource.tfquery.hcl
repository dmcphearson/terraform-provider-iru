# Every device on one blueprint, with config for import.
list "iru_device" "core" {
  provider         = iru
  include_resource = true
  config {
    blueprint_id = "00000000-0000-0000-0000-000000000000"
  }
}
