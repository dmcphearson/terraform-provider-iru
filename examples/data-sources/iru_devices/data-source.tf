# Every Mac on a blueprint.
data "iru_devices" "core_macs" {
  blueprint_id = iru_blueprint.core.id
  platform     = "Mac"
}

output "core_mac_serials" {
  value = data.iru_devices.core_macs.results[*].serial_number
}
