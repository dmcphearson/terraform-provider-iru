# Upload a local installer. It is re-uploaded only when the file's SHA256 changes.
resource "iru_custom_app" "swift_dialog" {
  name                = "swiftDialog"
  source              = "${path.module}/dialog-2.5.6.pkg"
  install_type        = "package"
  install_enforcement = "install_once"
  active              = true
}

# Or reference a binary uploaded out of band by its file_key.
resource "iru_custom_app" "legacy_tool" {
  name                = "Legacy Tool"
  file_key            = "tenants/.../library/custom_apps/legacy-tool_8a412a32.pkg"
  install_type        = "package"
  install_enforcement = "install_once"
}
