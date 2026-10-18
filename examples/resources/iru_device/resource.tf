# Devices can't be created: import an enrolled device (see Import below), then
# manage these fields.
resource "iru_device" "loaner" {
  asset_tag    = "IT-0042"
  user_id      = data.iru_user.alex.id
  blueprint_id = iru_blueprint.core.id
  tags         = ["loaner", "engineering"]
}
