# There is one routing setting per tenant; its ID is always blueprint_routing.
# Destroy only removes it from state.
terraform import iru_blueprint_routing.this blueprint_routing
