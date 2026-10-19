package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeTenant serves one object of each type, plus a second blueprint with no
// assigned items.
func fakeTenant(t *testing.T) *client.Client {
	t.Helper()
	pages := map[string]string{
		"/api/v1/devices":                          `[{"device_id":"d1","device_name":"Mac","serial_number":"C02","user":"","tags":["eng"]}]`,
		"/api/v1/blueprint-routing/":               `{"enrollment_code":{"code":"123456","is_active":true}}`,
		"/api/v1/devices/d1/notes":                 `[{"note_id":"n1","content":"<p>Loaner</p>","author":"Admin"}]`,
		"/api/v1/tags":                             `{"count":1,"results":[{"id":"t1","name":"Eng"}]}`,
		"/api/v1/library/custom-scripts":           `{"count":1,"results":[{"id":"s1","name":"Hello"}]}`,
		"/api/v1/library/custom-scripts/s1":        `{"id":"s1","name":"Hello","execution_frequency":"once","script":"echo hi","active":true}`,
		"/api/v1/library/custom-profiles":          `{"count":1,"results":[{"id":"p1","name":"Wifi"}]}`,
		"/api/v1/library/custom-profiles/p1":       `{"id":"p1","name":"Wifi","profile":"<plist/>","active":true,"runs_on_mac":true}`,
		"/api/v1/library/custom-apps":              `{"count":1,"results":[{"id":"a1","name":"Tool"}]}`,
		"/api/v1/library/custom-apps/a1":           `{"id":"a1","name":"Tool","file_key":"k1","install_type":"zip","install_enforcement":"install_once","active":true}`,
		"/api/v1/library/ipa-apps":                 `{"count":1,"results":[{"id":"i1","name":"Field"}]}`,
		"/api/v1/library/ipa-apps/i1":              `{"id":"i1","name":"Field","file_key":"k2","active":true,"runs_on_iphone":true}`,
		"/api/v1/blueprints":                       `{"count":2,"results":[{"id":"b1","name":"Core","type":"map"},{"id":"b2","name":"Lab","type":"map"}]}`,
		"/api/v1/blueprints/b1":                    `{"id":"b1","name":"Core","type":"map","enrollment_code":{"code":"123","is_active":false}}`,
		"/api/v1/blueprints/b2":                    `{"id":"b2","name":"Lab","type":"map"}`,
		"/api/v1/blueprints/b1/list-library-items": `{"count":1,"results":[{"id":"s1","name":"Hello"}]}`,
		"/api/v1/blueprints/b2/list-library-items": `{"count":0,"results":[]}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return client.New(srv.URL, "test-token", "test")
}

// runList runs a list resource against the fake tenant, with its managed
// resource's schemas, and returns the results.
func runList(t *testing.T, newList func() list.ListResource, filter *string) []list.ListResult {
	t.Helper()
	ctx := context.Background()
	c := fakeTenant(t)
	p := &iruProvider{}

	lr := newList()
	var meta resource.MetadataResponse
	lr.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "iru"}, &meta)
	lr.(list.ListResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: c}, &resource.ConfigureResponse{})

	var res resource.Resource
	for _, f := range p.Resources(ctx) {
		var m resource.MetadataResponse
		f().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "iru"}, &m)
		if m.TypeName == meta.TypeName {
			res = f()
		}
	}
	if res == nil {
		t.Fatalf("no managed resource named %s", meta.TypeName)
	}
	var rs resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &rs)
	var is resource.IdentitySchemaResponse
	res.(resource.ResourceWithIdentity).IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &is)

	var ls list.ListResourceSchemaResponse
	lr.ListResourceConfigSchema(ctx, list.ListResourceSchemaRequest{}, &ls)
	attrs := map[string]tftypes.Value{}
	for name := range ls.Schema.Attributes {
		var v any
		if filter != nil {
			v = *filter
		}
		attrs[name] = tftypes.NewValue(tftypes.String, v)
	}
	cfg := tfsdk.Config{Schema: ls.Schema, Raw: tftypes.NewValue(ls.Schema.Type().TerraformType(ctx), attrs)}

	var stream list.ListResultsStream
	lr.List(ctx, list.ListRequest{
		Config:                 cfg,
		IncludeResource:        true,
		Limit:                  100,
		ResourceSchema:         rs.Schema,
		ResourceIdentitySchema: is.IdentitySchema,
	}, &stream)
	if stream.Results == nil {
		t.Fatal("List set no results iterator")
	}
	var out []list.ListResult
	for r := range stream.Results {
		if r.Diagnostics.HasError() {
			t.Fatalf("%s: %v", meta.TypeName, r.Diagnostics)
		}
		out = append(out, r)
	}
	return out
}

func TestListResources(t *testing.T) {
	cases := []struct {
		newList     func() list.ListResource
		displayName string
		identityID  string // id identity, library_item_id for assignments, note_id for notes
	}{
		{NewTagListResource, "Eng", "t1"},
		{NewDeviceListResource, "Mac (C02)", "d1"},
		{NewBlueprintRoutingListResource, "Blueprint Routing", "blueprint_routing"},
		{NewDeviceNoteListResource, "Mac: Loaner", "n1"},
		{NewCustomScriptListResource, "Hello", "s1"},
		{NewCustomProfileListResource, "Wifi", "p1"},
		{NewCustomAppListResource, "Tool", "a1"},
		{NewInHouseAppListResource, "Field", "i1"},
		{NewBlueprintAssignmentListResource, "Core / Hello", "s1"},
	}
	for _, tc := range cases {
		t.Run(tc.displayName, func(t *testing.T) {
			results := runList(t, tc.newList, nil)
			if len(results) != 1 {
				t.Fatalf("got %d results, want 1", len(results))
			}
			r := results[0]
			if r.DisplayName != tc.displayName {
				t.Errorf("display name = %q, want %q", r.DisplayName, tc.displayName)
			}
			attr := map[string]string{"Core / Hello": "library_item_id", "Mac: Loaner": "note_id"}[tc.displayName]
			if attr == "" {
				attr = "id"
			}
			var got types.String
			r.Identity.GetAttribute(context.Background(), path.Root(attr), &got)
			if got.ValueString() != tc.identityID {
				t.Errorf("identity %s = %q, want %q", attr, got.ValueString(), tc.identityID)
			}
			var id types.String
			r.Resource.GetAttribute(context.Background(), path.Root("id"), &id)
			if id.IsNull() || id.ValueString() == "" {
				t.Error("resource id not set")
			}
		})
	}
}

func TestListResources_Blueprints(t *testing.T) {
	if got := runList(t, NewBlueprintListResource, nil); len(got) != 2 {
		t.Fatalf("got %d blueprints, want 2", len(got))
	}
	lab := "Lab"
	got := runList(t, NewBlueprintListResource, &lab)
	if len(got) != 1 || got[0].DisplayName != "Lab" {
		t.Fatalf("name filter: got %d results, want only Lab", len(got))
	}
}

// Profiles need profile_file in generated config, so the listed resource carries
// the profile Iru holds.
func TestListResources_ProfileCarriesProfileFile(t *testing.T) {
	r := runList(t, NewCustomProfileListResource, nil)[0]
	var pf types.String
	r.Resource.GetAttribute(context.Background(), path.Root("profile_file"), &pf)
	if pf.ValueString() != "<plist/>" {
		t.Errorf("profile_file = %q, want <plist/>", pf.ValueString())
	}
}
