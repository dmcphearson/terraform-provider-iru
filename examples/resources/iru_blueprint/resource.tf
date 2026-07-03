resource "iru_blueprint" "core" {
  name        = "Core"
  description = "Baseline blueprint"
  icon        = "ss-files"
  color       = "aqua-500"
  type        = "classic"

  enrollment_code = {
    is_active = true
  }
}
