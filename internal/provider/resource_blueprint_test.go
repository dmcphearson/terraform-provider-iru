package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCheckBlueprintType(t *testing.T) {
	cases := []struct {
		name    string
		prior   types.String
		planned types.String
		wantErr bool
	}{
		{"create map", types.StringNull(), types.StringValue("map"), false},
		{"create unset", types.StringNull(), types.StringUnknown(), false},
		{"create classic", types.StringNull(), types.StringValue("classic"), true},
		{"keep imported classic", types.StringValue("classic"), types.StringValue("classic"), false},
		{"replace map with classic", types.StringValue("map"), types.StringValue("classic"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			checkBlueprintType(tc.prior, tc.planned, &diags)
			if diags.HasError() != tc.wantErr {
				t.Errorf("error = %v, want %v: %v", diags.HasError(), tc.wantErr, diags)
			}
		})
	}
}
