package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// readDataSource runs Read with the given config attributes; the rest are null.
func readDataSource(t *testing.T, d datasource.DataSource, srv *httptest.Server, set map[string]any) (tfsdk.State, bool) {
	t.Helper()
	ctx := context.Background()
	d.(datasource.DataSourceWithConfigure).Configure(ctx,
		datasource.ConfigureRequest{ProviderData: client.New(srv.URL, "test-token", "test")}, &datasource.ConfigureResponse{})
	var sr datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &sr)

	objType := sr.Schema.Type().TerraformType(ctx).(tftypes.Object)
	nulls := map[string]tftypes.Value{}
	for k, at := range objType.AttributeTypes {
		nulls[k] = tftypes.NewValue(at, nil)
	}
	cfg := tfsdk.Config{Schema: sr.Schema, Raw: tftypes.NewValue(objType, nulls)}
	state := tfsdk.State{Schema: sr.Schema, Raw: cfg.Raw}
	for k, v := range set {
		if diags := state.SetAttribute(ctx, path.Root(k), v); diags.HasError() {
			t.Fatal(diags)
		}
	}
	cfg.Raw = state.Raw
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: sr.Schema, Raw: cfg.Raw}}
	d.Read(ctx, datasource.ReadRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Log(resp.Diagnostics)
	}
	return resp.State, !resp.Diagnostics.HasError()
}

func serve(t *testing.T, pages map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// The API matches serial_number by substring, so a lookup of ABC1 also returns
// ABC12; the data source must pick the exact match.
func TestDeviceDataSource_ExactSerial(t *testing.T) {
	srv := serve(t, map[string]string{
		"/api/v1/devices": `[
			{"device_id":"d2","serial_number":"ABC12","user":""},
			{"device_id":"d1","serial_number":"ABC1","user":{"id":"u1","email":"a@example.com","name":"A"},"tags":["eng"]}]`,
	})
	state, ok := readDataSource(t, NewDeviceDataSource(), srv, map[string]any{"serial_number": "ABC1"})
	if !ok {
		t.Fatal("read failed")
	}
	var got deviceFields
	state.Get(context.Background(), &got)
	if got.ID.ValueString() != "d1" || got.UserEmail.ValueString() != "a@example.com" {
		t.Errorf("got id %q user %q, want d1 and a@example.com", got.ID.ValueString(), got.UserEmail.ValueString())
	}

	if _, ok := readDataSource(t, NewDeviceDataSource(), srv, map[string]any{"serial_number": "ABC"}); ok {
		t.Error("partial serial ABC matched; want not found")
	}
}

func TestDevicesDataSource_UnassignedDevice(t *testing.T) {
	srv := serve(t, map[string]string{
		"/api/v1/devices": `[{"device_id":"d2","serial_number":"ABC12","user":""}]`,
	})
	state, ok := readDataSource(t, NewDevicesDataSource(), srv, nil)
	if !ok {
		t.Fatal("read failed")
	}
	var got devicesModel
	state.Get(context.Background(), &got)
	if len(got.Results) != 1 || !got.Results[0].UserEmail.IsNull() {
		t.Errorf("results = %+v, want one device with null user_email", got.Results)
	}
}

func TestDeviceDetailsDataSource_ReturnsJSON(t *testing.T) {
	srv := serve(t, map[string]string{
		"/api/v1/devices/d1/details": `{"general":{"device_name":"Mac"}}`,
	})
	state, ok := readDataSource(t, NewDeviceDetailsDataSource(), srv, map[string]any{"id": "d1"})
	if !ok {
		t.Fatal("read failed")
	}
	var j types.String
	state.GetAttribute(context.Background(), path.Root("json"), &j)
	if j.ValueString() != `{"general":{"device_name":"Mac"}}` {
		t.Errorf("json = %s", j.ValueString())
	}
}

func TestUserDataSource_EmailCaseInsensitive(t *testing.T) {
	srv := serve(t, map[string]string{
		"/api/v1/users": `{"next":null,"results":[
			{"id":"u9","email":"other@example.com"},
			{"id":"u1","email":"A.User@Example.com","department":null,"integration":{"uuid":"i1","name":"Dir","type":"office365"}}]}`,
	})
	state, ok := readDataSource(t, NewUserDataSource(), srv, map[string]any{"email": "a.user@example.com"})
	if !ok {
		t.Fatal("read failed")
	}
	var got userFields
	state.Get(context.Background(), &got)
	if got.ID.ValueString() != "u1" || got.IntegrationType.ValueString() != "office365" {
		t.Errorf("got id %q integration %q, want u1 and office365", got.ID.ValueString(), got.IntegrationType.ValueString())
	}
}
