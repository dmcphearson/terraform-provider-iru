# The routing setting always exists: import it (see Import below), then manage
# its enrollment code.
resource "iru_blueprint_routing" "this" {
  enrollment_code = {
    is_active = true
  }
}
