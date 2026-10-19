package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func routingModel(code attr.Value, active attr.Value) blueprintRoutingModel {
	obj, _ := types.ObjectValue(enrollmentCodeAttrTypes, map[string]attr.Value{"code": code, "is_active": active})
	return blueprintRoutingModel{ID: types.StringValue(blueprintRoutingID), EnrollmentCode: obj}
}

// The routing code is tenant-wide, so an update sends only what changed.
func TestBlueprintRoutingPatch_OnlyChangedKeys(t *testing.T) {
	state := routingModel(types.StringValue("123456"), types.BoolValue(true))
	cases := []struct {
		name string
		plan blueprintRoutingModel
		want map[string]any
	}{
		{"no change", state, map[string]any{}},
		{"deactivate", routingModel(types.StringValue("123456"), types.BoolValue(false)), map[string]any{"is_active": false}},
		{"new code", routingModel(types.StringValue("654321"), types.BoolValue(true)), map[string]any{"code": "654321"}},
		{"unknown left alone", routingModel(types.StringUnknown(), types.BoolUnknown()), map[string]any{}},
		{"block omitted", blueprintRoutingModel{EnrollmentCode: types.ObjectNull(enrollmentCodeAttrTypes)}, map[string]any{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := blueprintRoutingPatch(context.Background(), state, tc.plan)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("patch = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// The setting always exists: Create must refuse, and Delete must not call the API.
func TestBlueprintRoutingResource_NoCreateNoDelete(t *testing.T) {
	ctx := context.Background()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	r := &blueprintRoutingResource{client: client.New(srv.URL, "test-token", "test")}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	empty := tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)

	cresp := &resource.CreateResponse{State: tfsdk.State{Schema: sr.Schema, Raw: empty}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: sr.Schema, Raw: empty}}, cresp)
	if !cresp.Diagnostics.HasError() {
		t.Error("Create succeeded; want an error")
	}

	state := tfsdk.State{Schema: sr.Schema, Raw: empty}
	m := routingModel(types.StringValue("123456"), types.BoolValue(true))
	if d := state.Set(ctx, &m); d.HasError() {
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

func TestBlueprintRoutingActivityDataSource(t *testing.T) {
	srv := serve(t, map[string]string{
		"/api/v1/blueprint-routing/activity": `{"count":2,"next":null,"results":[
			{"id":2,"activity_type":"routing_config_edited","user":{"id":"u1","email":"admin@example.com"},"device_id":null,"details":{}},
			{"id":1,"activity_type":"enrollment","user":null,"device_id":"d1","details":{"enrollment_type":"DEP"}}]}`,
	})
	state, ok := readDataSource(t, NewBlueprintRoutingActivityDataSource(), srv, nil)
	if !ok {
		t.Fatal("read failed")
	}
	var got blueprintRoutingActivityModel
	state.Get(context.Background(), &got)
	if len(got.Results) != 2 {
		t.Fatalf("got %d events, want 2", len(got.Results))
	}
	edit, enroll := got.Results[0], got.Results[1]
	if edit.UserEmail.ValueString() != "admin@example.com" || !edit.DeviceID.IsNull() {
		t.Errorf("config event = %+v", edit)
	}
	if !enroll.UserID.IsNull() || enroll.DeviceID.ValueString() != "d1" || enroll.Details.ValueString() != `{"enrollment_type":"DEP"}` {
		t.Errorf("device event = %+v", enroll)
	}
}
