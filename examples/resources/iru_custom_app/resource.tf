# Metadata-only: the installer binary is uploaded out of band; reference it by
# file_key. Update file_key after uploading a new binary to roll it out.
resource "iru_custom_app" "swift_dialog" {
  name                = "swiftDialog"
  file_key            = "tenants/.../library/custom_apps/dialog-2.5.6_8a412a32.pkg"
  install_type        = "package"
  install_enforcement = "install_once"
  active              = true
}
