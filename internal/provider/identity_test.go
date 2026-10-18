package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// emptyStateAndIdentity returns a null state and a null identity for r, as the
// framework hands them to ImportState and Read.
func emptyStateAndIdentity(t *testing.T, r resource.Resource) (tfsdk.State, *tfsdk.ResourceIdentity) {
	t.Helper()
	ctx := context.Background()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	var ir resource.IdentitySchemaResponse
	r.(resource.ResourceWithIdentity).IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &ir)
	state := tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)}
	ident := &tfsdk.ResourceIdentity{Schema: ir.IdentitySchema, Raw: tftypes.NewValue(ir.IdentitySchema.Type().TerraformType(ctx), nil)}
	return state, ident
}

// Every resource must expose a valid identity schema; the framework validates them here.
func TestGetResourceIdentitySchemas(t *testing.T) {
	srv, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.GetResourceIdentitySchemas(context.Background(), &tfprotov6.GetResourceIdentitySchemasRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("diagnostic: %s: %s", d.Summary, d.Detail)
	}
	want := []string{"iru_custom_script", "iru_custom_profile", "iru_tag", "iru_custom_app", "iru_blueprint", "iru_blueprint_assignment"}
	for _, name := range want {
		if _, ok := resp.IdentitySchemas[name]; !ok {
			t.Errorf("%s has no identity schema", name)
		}
	}
}

// Import blocks with identity = { id = ... } must land the id in state.
func TestImportState_ByIdentity(t *testing.T) {
	ctx := context.Background()
	for _, newRes := range []func() resource.Resource{
		NewCustomScriptResource, NewCustomProfileResource, NewTagResource, NewCustomAppResource, NewBlueprintResource,
	} {
		r := newRes()
		state, ident := emptyStateAndIdentity(t, r)
		if d := ident.Set(ctx, idIdentityModel{ID: types.StringValue("abc")}); d.HasError() {
			t.Fatal(d)
		}
		resp := &resource.ImportStateResponse{State: state, Identity: ident}
		r.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{Identity: ident}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%T: %v", r, resp.Diagnostics)
		}
		var id types.String
		resp.State.GetAttribute(ctx, path.Root("id"), &id)
		if id.ValueString() != "abc" {
			t.Errorf("%T: state id = %q, want abc", r, id.ValueString())
		}
	}
}

// After `terraform import <id>` the identity is null. Read must not depend on it and
// must populate it (the crash reported against another Iru provider).
func TestTagRead_PopulatesNullIdentity(t *testing.T) {
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":1,"results":[{"id":"abc","name":"prod"}]}`))
	}))
	t.Cleanup(srv.Close)

	r := &tagResource{client: client.New(srv.URL, "test-token", "test")}
	state, ident := emptyStateAndIdentity(t, r)
	if d := state.SetAttribute(ctx, path.Root("id"), "abc"); d.HasError() {
		t.Fatal(d)
	}
	resp := &resource.ReadResponse{State: state, Identity: ident}
	r.Read(ctx, resource.ReadRequest{State: state, Identity: ident}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var got idIdentityModel
	resp.Identity.Get(ctx, &got)
	if got.ID.ValueString() != "abc" {
		t.Errorf("identity id = %q, want abc", got.ID.ValueString())
	}
	var name types.String
	resp.State.GetAttribute(ctx, path.Root("name"), &name)
	if name.ValueString() != "prod" {
		t.Errorf("name = %q, want prod", name.ValueString())
	}
}

func TestBlueprintAssignmentImport(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		id     string
		ident  *blueprintAssignmentIdentityModel
		wantID string
	}{
		{name: "import id", id: "bp:item", wantID: "bp:item"},
		{name: "import id with node", id: "bp:item:node", wantID: "bp:item:node"},
		{name: "identity", ident: &blueprintAssignmentIdentityModel{
			BlueprintID: types.StringValue("bp"), LibraryItemID: types.StringValue("item"), AssignmentNodeID: types.StringNull(),
		}, wantID: "bp:item"},
		{name: "identity with node", ident: &blueprintAssignmentIdentityModel{
			BlueprintID: types.StringValue("bp"), LibraryItemID: types.StringValue("item"), AssignmentNodeID: types.StringValue("node"),
		}, wantID: "bp:item:node"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &blueprintAssignmentResource{}
			state, ident := emptyStateAndIdentity(t, r)
			if tc.ident != nil {
				if d := ident.Set(ctx, tc.ident); d.HasError() {
					t.Fatal(d)
				}
			}
			resp := &resource.ImportStateResponse{State: state, Identity: ident}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tc.id, Identity: ident}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var m blueprintAssignmentModel
			resp.State.Get(ctx, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("id = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.BlueprintID.ValueString() != "bp" || m.LibraryItemID.ValueString() != "item" {
				t.Errorf("state = %+v", m)
			}
		})
	}
}
