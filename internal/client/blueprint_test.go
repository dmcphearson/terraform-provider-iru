package client

import (
	"context"
	"io"
	"net/http"
	"testing"
)

// Assign and remove are distinct POST endpoints per the Iru API docs:
// .../assign-library-item and .../remove-library-item.
func TestAssignAndRemoveUseCorrectEndpoints(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))

	if err := c.AssignLibraryItem(context.Background(), "bp1", "li1", "node1"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if gotMethod != "POST" || gotPath != "/api/v1/blueprints/bp1/assign-library-item" {
		t.Errorf("assign = %s %s, want POST .../assign-library-item", gotMethod, gotPath)
	}
	if gotBody != `{"library_item_id":"li1","assignment_node_id":"node1"}` {
		t.Errorf("assign body = %s", gotBody)
	}

	if err := c.RemoveLibraryItem(context.Background(), "bp1", "li1", ""); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if gotMethod != "POST" || gotPath != "/api/v1/blueprints/bp1/remove-library-item" {
		t.Errorf("remove = %s %s, want POST .../remove-library-item", gotMethod, gotPath)
	}
	// assignment_node_id omitempty: absent when empty.
	if gotBody != `{"library_item_id":"li1"}` {
		t.Errorf("remove body = %s, want no node id", gotBody)
	}
}

func TestIsLibraryItemAssigned(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"count":2,"results":[{"id":"li1","name":"a"},{"id":"li2","name":"b"}]}`))
	}))
	ok, err := c.IsLibraryItemAssigned(context.Background(), "bp1", "li2")
	if err != nil || !ok {
		t.Errorf("want assigned=true, got %v err=%v", ok, err)
	}
	no, err := c.IsLibraryItemAssigned(context.Background(), "bp1", "missing")
	if err != nil || no {
		t.Errorf("want assigned=false, got %v err=%v", no, err)
	}
}

// The custom-app create/update body must be urlencoded with the expected fields.
func TestCustomAppFormEncoding(t *testing.T) {
	var gotCT, gotBody string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Write([]byte(`{"id":"a1","name":"App"}`))
	}))
	_, err := c.CreateCustomApp(context.Background(), CustomApp{
		Name: "App", FileKey: "k", InstallType: "zip", InstallEnforcement: "install_once",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if gotCT != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %s", gotCT)
	}
	for _, want := range []string{"name=App", "file_key=k", "install_type=zip", "install_enforcement=install_once"} {
		if !contains(gotBody, want) {
			t.Errorf("body %q missing %q", gotBody, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
