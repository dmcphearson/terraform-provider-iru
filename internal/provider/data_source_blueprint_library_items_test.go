package provider

import (
	"context"
	"testing"
)

func TestBlueprintLibraryItemsDataSource(t *testing.T) {
	srv := serve(t, map[string]string{
		"/api/v1/blueprints/b1/list-library-items": `{"count":2,"next":null,"results":[{"id":"s1","name":"Hello"},{"id":"p1","name":"Wifi"}]}`,
		"/api/v1/blueprints/b2/list-library-items": `{"count":0,"next":null,"results":[]}`,
	})
	state, ok := readDataSource(t, NewBlueprintLibraryItemsDataSource(), srv, map[string]any{"blueprint_id": "b1"})
	if !ok {
		t.Fatal("read failed")
	}
	var got blueprintLibraryItemsModel
	state.Get(context.Background(), &got)
	if len(got.Results) != 2 || got.Results[0].ID.ValueString() != "s1" || got.Results[1].Name.ValueString() != "Wifi" {
		t.Errorf("results = %+v", got.Results)
	}

	// An empty blueprint gives an empty list, not null.
	state, _ = readDataSource(t, NewBlueprintLibraryItemsDataSource(), srv, map[string]any{"blueprint_id": "b2"})
	state.Get(context.Background(), &got)
	if got.Results == nil || len(got.Results) != 0 {
		t.Errorf("empty blueprint results = %#v, want []", got.Results)
	}

	if _, ok := readDataSource(t, NewBlueprintLibraryItemsDataSource(), srv, map[string]any{"blueprint_id": "missing"}); ok {
		t.Error("unknown blueprint read succeeded; want an error")
	}
}
