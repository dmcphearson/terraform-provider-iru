data "iru_device_details" "loaner" {
  id = data.iru_device.loaner.id
}

locals {
  loaner_details = jsondecode(data.iru_device_details.loaner.json)
}
