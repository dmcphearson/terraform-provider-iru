package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateSelfService(t *testing.T) {
	cases := []struct {
		name        string
		enforcement types.String
		show        types.Bool
		recommended types.Bool
		wantErr     bool
	}{
		{"enforced, Self Service off", types.StringValue("once"), types.BoolValue(false), types.BoolNull(), false},
		{"no_enforcement, Self Service on", types.StringValue("no_enforcement"), types.BoolValue(true), types.BoolNull(), false},
		{"no_enforcement, Self Service off", types.StringValue("no_enforcement"), types.BoolValue(false), types.BoolNull(), true},
		{"no_enforcement, Self Service unset", types.StringValue("no_enforcement"), types.BoolNull(), types.BoolNull(), true},
		{"no_enforcement, Self Service unknown", types.StringValue("no_enforcement"), types.BoolUnknown(), types.BoolNull(), false},
		{"recommended, Self Service off", types.StringValue("once"), types.BoolNull(), types.BoolValue(true), true},
		{"recommended, Self Service on", types.StringValue("once"), types.BoolValue(true), types.BoolValue(true), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			validateSelfService("execution_frequency", tc.enforcement, tc.show, tc.recommended, &diags)
			if diags.HasError() != tc.wantErr {
				t.Errorf("error = %v, want %v: %v", diags.HasError(), tc.wantErr, diags)
			}
		})
	}
}
