package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestBlueprintsDataSource_FiltersByName(t *testing.T) {
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"count":2,"results":[
			{"id":"a","name":"Core","type":"classic","description":"","computers_count":3},
			{"id":"b","name":"Lab","type":"map","description":"test","computers_count":0}]}`))
	}))
	t.Cleanup(srv.Close)

	d := &blueprintsDataSource{client: client.New(srv.URL, "test-token", "test")}
	var sr datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &sr)
	// Build the config value { name = "Lab" } via a State, which has a typed setter.
	state := tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &blueprintsModel{Name: types.StringValue("Lab")}); diags.HasError() {
		t.Fatal(diags)
	}
	cfg := tfsdk.Config{Schema: sr.Schema, Raw: state.Raw}

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: sr.Schema, Raw: cfg.Raw}}
	d.Read(ctx, datasource.ReadRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var got blueprintsModel
	resp.State.Get(ctx, &got)
	if len(got.Results) != 1 || got.Results[0].ID.ValueString() != "b" || got.Results[0].Type.ValueString() != "map" {
		t.Errorf("results = %+v, want only blueprint b", got.Results)
	}
}
