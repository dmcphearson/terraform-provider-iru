package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Notes written in the Iru UI come back as HTML. Config text that matches apart
// from markup must not drift; a real change and an import take the API's text.
func TestApplyDeviceNote_Content(t *testing.T) {
	cases := []struct {
		name, config, api, want string
		null                    bool
	}{
		{name: "same", config: "Loaner", api: "Loaner", want: "Loaner"},
		{name: "html wrapped", config: "Loaner for QA", api: "<p>Loaner for QA&nbsp;</p>", want: "Loaner for QA"},
		{name: "changed in Iru", config: "Loaner", api: "<p>Returned</p>", want: "<p>Returned</p>"},
		{name: "import", null: true, api: "<p>Loaner</p>", want: "<p>Loaner</p>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := deviceNoteModel{DeviceID: types.StringValue("d1"), NoteID: types.StringValue("n1"), Content: types.StringValue(tc.config)}
			if tc.null {
				m.Content = types.StringNull()
			}
			applyDeviceNote(&m, &client.DeviceNote{ID: "n1", Content: tc.api})
			if m.Content.ValueString() != tc.want {
				t.Errorf("content = %q, want %q", m.Content.ValueString(), tc.want)
			}
			if m.ID.ValueString() != "d1:n1" {
				t.Errorf("id = %q, want d1:n1", m.ID.ValueString())
			}
		})
	}
}

// The note's lifecycle must only ever touch that device's notes endpoints, never
// the device itself.
func TestDeviceNoteResource_TouchesOnlyNotes(t *testing.T) {
	ctx := context.Background()
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, r.Method+" "+r.URL.Path+" "+string(body))
		_, _ = w.Write([]byte(`{"note_id":"n1","content":"hi","author":"API","created_at":"t0","updated_at":"t1"}`))
	}))
	t.Cleanup(srv.Close)
	r := &deviceNoteResource{client: client.New(srv.URL, "test-token", "test")}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	empty := tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)
	unknown := types.StringUnknown()

	plan := tfsdk.Plan{Schema: sr.Schema, Raw: empty}
	if d := plan.Set(ctx, &deviceNoteModel{ID: unknown, DeviceID: types.StringValue("d1"), NoteID: unknown,
		Content: types.StringValue("hi"), Author: unknown, CreatedAt: unknown, UpdatedAt: unknown}); d.HasError() {
		t.Fatal(d)
	}
	cresp := &resource.CreateResponse{State: tfsdk.State{Schema: sr.Schema, Raw: empty}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cresp)
	if cresp.Diagnostics.HasError() {
		t.Fatal(cresp.Diagnostics)
	}
	var created deviceNoteModel
	cresp.State.Get(ctx, &created)
	if created.NoteID.ValueString() != "n1" || created.ID.ValueString() != "d1:n1" {
		t.Errorf("created = %+v", created)
	}

	dresp := &resource.DeleteResponse{State: cresp.State}
	r.Delete(ctx, resource.DeleteRequest{State: cresp.State}, dresp)
	if dresp.Diagnostics.HasError() {
		t.Fatal(dresp.Diagnostics)
	}

	want := []string{
		`POST /api/v1/devices/d1/notes {"content":"hi"}`,
		"DELETE /api/v1/devices/d1/notes/n1 ",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
