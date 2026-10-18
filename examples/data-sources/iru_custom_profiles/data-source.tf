data "iru_custom_profiles" "wifi" {
  name = "Wi-Fi"
}

# Reference the matching ID:
#   one(data.iru_custom_profiles.wifi.results).id
