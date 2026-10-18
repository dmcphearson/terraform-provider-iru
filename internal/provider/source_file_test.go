package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNeedsUpload(t *testing.T) {
	cases := []struct {
		name         string
		sourceSHA256 types.String
		apiSHA256    types.String
		want         bool
	}{
		{"unchanged file", types.StringValue("aaa"), types.StringValue("aaa"), false},
		{"new file", types.StringValue("aaa"), types.StringValue("aaa"), true},
		{"imported, binary matches", types.StringNull(), types.StringValue("bbb"), false},
		{"imported, binary differs", types.StringNull(), types.StringValue("ccc"), true},
	}
	sums := []string{"aaa", "bbb", "bbb", "bbb"}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := needsUpload(tc.sourceSHA256, tc.apiSHA256, sums[i]); got != tc.want {
				t.Errorf("needsUpload = %v, want %v", got, tc.want)
			}
		})
	}
}
