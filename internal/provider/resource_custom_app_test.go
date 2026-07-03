package provider

import (
	"testing"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The API trims trailing whitespace from script bodies (audit/preinstall/postinstall),
// just like custom_script. When the configured value differs from the server value only
// by trailing whitespace, the configured value must be kept in state so the apply result
// matches the plan (otherwise: "Provider produced inconsistent result after apply").
func TestApplyAppResponse_PreservesScriptTrailingNewline(t *testing.T) {
	prior := customAppModel{
		PreinstallScript:  types.StringValue("#!/bin/bash\necho hi\n"),
		PostinstallScript: types.StringValue("echo bye\n"),
		AuditScript:       types.StringValue("audit\n"),
	}
	out := &client.CustomApp{
		ID:                "1",
		PreinstallScript:  "#!/bin/bash\necho hi",
		PostinstallScript: "echo bye",
		AuditScript:       "audit",
	}

	var dst customAppModel
	applyAppResponse(&dst, out, prior)

	if dst.PreinstallScript.ValueString() != "#!/bin/bash\necho hi\n" {
		t.Errorf("preinstall_script = %q, want configured value with trailing newline preserved", dst.PreinstallScript.ValueString())
	}
	if dst.PostinstallScript.ValueString() != "echo bye\n" {
		t.Errorf("postinstall_script = %q, want configured value preserved", dst.PostinstallScript.ValueString())
	}
	if dst.AuditScript.ValueString() != "audit\n" {
		t.Errorf("audit_script = %q, want configured value preserved", dst.AuditScript.ValueString())
	}
}

// When a script body genuinely changed (not just trailing whitespace), adopt the server value.
func TestApplyAppResponse_AdoptsRealScriptChange(t *testing.T) {
	prior := customAppModel{PreinstallScript: types.StringValue("echo old\n")}
	out := &client.CustomApp{ID: "1", PreinstallScript: "echo new"}

	var dst customAppModel
	applyAppResponse(&dst, out, prior)

	if dst.PreinstallScript.ValueString() != "echo new" {
		t.Errorf("preinstall_script = %q, want server value adopted on real change", dst.PreinstallScript.ValueString())
	}
}

// A script left unset in config (null prior) adopts the API-returned "" so it settles
// on a known value and re-plan is stable.
func TestApplyAppResponse_UnsetScriptAdoptsEmpty(t *testing.T) {
	prior := customAppModel{AuditScript: types.StringNull()}
	out := &client.CustomApp{ID: "1", AuditScript: ""}

	var dst customAppModel
	applyAppResponse(&dst, out, prior)

	if dst.AuditScript.IsNull() || dst.AuditScript.ValueString() != "" {
		t.Errorf("audit_script = %v, want empty string adopted", dst.AuditScript)
	}
}
