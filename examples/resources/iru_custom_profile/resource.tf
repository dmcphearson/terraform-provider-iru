resource "iru_custom_profile" "wifi" {
  name         = "Corp Wi-Fi"
  profile_file = file("${path.module}/corp-wifi.mobileconfig")
  active       = true
  runs_on_mac  = true
}
