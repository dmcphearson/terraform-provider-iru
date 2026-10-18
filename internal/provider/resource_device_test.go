package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func tagSet(tags ...string) types.Set {
	v := make([]types.String, len(tags))
	for i, t := range tags {
		v[i] = types.StringValue(t)
	}
	s, _ := types.SetValueFrom(context.Background(), types.StringType, v)
	return s
}

// An update must send only what changed, so a plan touching one attribute can't
// rewrite the others on a real device.
func TestDevicePatch_OnlyChangedFields(t *testing.T) {
	state := deviceModel{
		AssetTag:    types.StringValue("A1"),
		UserID:      types.StringValue("u1"),
		BlueprintID: types.StringValue("bp1"),
		Tags:        tagSet("eng", "laptop"),
	}
	cases := []struct {
		name string
		edit func(*deviceModel)
		want map[string]any
	}{
		{"no change", func(*deviceModel) {}, map[string]any{}},
		{"asset tag", func(m *deviceModel) { m.AssetTag = types.StringValue("A2") }, map[string]any{"asset_tag": "A2"}},
		{"clear asset tag", func(m *deviceModel) { m.AssetTag = types.StringValue("") }, map[string]any{"asset_tag": nil}},
		{"clear user", func(m *deviceModel) { m.UserID = types.StringValue("") }, map[string]any{"user": nil}},
		{"blueprint", func(m *deviceModel) { m.BlueprintID = types.StringValue("bp2") }, map[string]any{"blueprint_id": "bp2"}},
		{"tags reordered", func(m *deviceModel) { m.Tags = tagSet("laptop", "eng") }, map[string]any{}},
		{"tags changed", func(m *deviceModel) { m.Tags = tagSet("eng") }, map[string]any{"tags": []string{"eng"}}},
		{"unknown left alone", func(m *deviceModel) { m.UserID = types.StringUnknown(); m.Tags = types.SetUnknown(types.StringType) }, map[string]any{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := state
			tc.edit(&plan)
			got, diags := devicePatch(context.Background(), state, plan)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("patch = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// Devices enroll through Iru; Create must refuse, and Delete must not call the API.
func TestDeviceResource_NoCreateNoDelete(t *testing.T) {
	ctx := context.Background()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	r := &deviceResource{client: client.New(srv.URL, "test-token", "test")}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	empty := tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)

	cresp := &resource.CreateResponse{State: tfsdk.State{Schema: sr.Schema, Raw: empty}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: sr.Schema, Raw: empty}}, cresp)
	if !cresp.Diagnostics.HasError() {
		t.Error("Create succeeded; want an error")
	}

	state := tfsdk.State{Schema: sr.Schema, Raw: empty}
	if d := state.Set(ctx, &deviceModel{ID: types.StringValue("d1"), Tags: tagSet()}); d.HasError() {
		t.Fatal(d)
	}
	dresp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dresp)
	if dresp.Diagnostics.HasError() {
		t.Error(dresp.Diagnostics)
	}
	if calls != 0 {
		t.Errorf("made %d API calls; want none", calls)
	}
}
