resource "iru_blueprint" "core" {
  name        = "Core"
  description = "Baseline blueprint"
  icon        = "ss-files"
  color       = "aqua-500"
  type        = "map"

  enrollment_code = {
    is_active = true
  }
}
