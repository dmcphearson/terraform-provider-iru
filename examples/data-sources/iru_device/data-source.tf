data "iru_device" "loaner" {
  serial_number = "C02XXXXXXXXX"
}

output "loaner_user" {
  value = data.iru_device.loaner.user_email
}
