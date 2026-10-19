resource "iru_device_note" "loaner" {
  device_id = iru_device.loaner.id
  content   = "Loaner for the QA team. Return to IT by end of quarter."
}
