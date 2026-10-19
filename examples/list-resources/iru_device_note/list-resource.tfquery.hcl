# Every note on one device, with config for import.
list "iru_device_note" "loaner" {
  provider         = iru
  include_resource = true
  config {
    device_id = "00000000-0000-0000-0000-000000000000"
  }
}
