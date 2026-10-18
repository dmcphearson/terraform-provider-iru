package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// validateSelfService catches two Self Service rules the API enforces only at apply:
// an item that is not enforced must be offered in Self Service, and
// self_service_recommended needs Self Service on. An unset show_in_self_service is
// sent as false, so it counts as off. Unknown values are skipped.
func validateSelfService(enforcementAttr string, enforcement types.String, show, recommended types.Bool, diags *diag.Diagnostics) {
	if show.IsUnknown() {
		return
	}
	if enforcement.ValueString() == "no_enforcement" && !show.ValueBool() {
		diags.AddAttributeError(path.Root("show_in_self_service"), "Self Service required",
			fmt.Sprintf("Iru requires show_in_self_service = true when %s is no_enforcement.", enforcementAttr))
	}
	if recommended.ValueBool() && !show.ValueBool() {
		diags.AddAttributeError(path.Root("self_service_recommended"), "Self Service required",
			"Iru accepts self_service_recommended only when show_in_self_service = true.")
	}
}
