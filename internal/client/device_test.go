package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// Devices come back as a bare array, 300 per page, and an unassigned device has
// "user": "" instead of an object.
func TestListDevices_PagesAndToleratesEmptyUser(t *testing.T) {
	var gotFilter string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilter = r.URL.Query().Get("blueprint_id")
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		n := 300
		if offset >= 300 {
			n = 2
		}
		_, _ = w.Write([]byte("["))
		for i := 0; i < n; i++ {
			if i > 0 {
				_, _ = w.Write([]byte(","))
			}
			user := `""`
			if i == 0 {
				user = `{"id":"u1","email":"a@example.com","name":"A","is_archived":false}`
			}
			_, _ = fmt.Fprintf(w, `{"device_id":"d%d","device_name":"Mac %d","user":%s,"tags":["x"]}`, offset+i, offset+i, user)
		}
		_, _ = w.Write([]byte("]"))
	}))

	devices, err := c.ListDevices(context.Background(), DeviceFilter{BlueprintID: "bp1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 302 {
		t.Fatalf("got %d devices, want 302", len(devices))
	}
	if gotFilter != "bp1" {
		t.Errorf("blueprint_id filter = %q, want bp1", gotFilter)
	}
	if devices[0].User == nil || devices[0].User.Email != "a@example.com" {
		t.Errorf("first device user = %+v, want a@example.com", devices[0].User)
	}
	if devices[1].User != nil {
		t.Errorf("unassigned device user = %+v, want nil", devices[1].User)
	}
}

// Users page with a cursor in an absolute next URL, which may name a different
// host than the API URL; only its path and query are followed.
func TestListUsers_FollowsCursor(t *testing.T) {
	var calls int
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("cursor") == "" {
			_, _ = w.Write([]byte(`{"next":"https://elsewhere.example.com/api/v1/users?cursor=p2","results":[{"id":"u1","email":"a@example.com","integration":{"uuid":"i1","name":"Dir","type":"office365"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"next":null,"results":[{"id":"u2","email":"b@example.com","department":null,"integration":null}]}`))
	}))

	users, err := c.ListUsers(context.Background(), UserFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(users) != 2 {
		t.Fatalf("calls = %d, users = %d, want 2 and 2", calls, len(users))
	}
	if users[0].Integration == nil || users[0].Integration.UUID != "i1" || users[1].Integration != nil {
		t.Errorf("integrations = %+v, %+v", users[0].Integration, users[1].Integration)
	}
}

// Notes live under their device; create and update send only the content.
func TestDeviceNotes_Paths(t *testing.T) {
	var got []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, r.Method+" "+r.URL.Path+" "+string(body))
		if r.Method == "GET" && r.URL.Path == "/api/v1/devices/d1/notes" {
			_, _ = w.Write([]byte(`[{"note_id":"n1","content":"<p>hi</p>"}]`))
			return
		}
		_, _ = w.Write([]byte(`{"note_id":"n1","content":"hi","author":"API"}`))
	}))
	ctx := context.Background()
	notes, err := c.ListDeviceNotes(ctx, "d1")
	if err != nil || len(notes) != 1 || notes[0].ID != "n1" {
		t.Fatalf("list = %+v, %v", notes, err)
	}
	if n, err := c.CreateDeviceNote(ctx, "d1", "hi"); err != nil || n.ID != "n1" || n.Author != "API" {
		t.Fatalf("create = %+v, %v", n, err)
	}
	if _, err := c.GetDeviceNote(ctx, "d1", "n1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateDeviceNote(ctx, "d1", "n1", "bye"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteDeviceNote(ctx, "d1", "n1"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /api/v1/devices/d1/notes ",
		`POST /api/v1/devices/d1/notes {"content":"hi"}`,
		"GET /api/v1/devices/d1/notes/n1 ",
		`PATCH /api/v1/devices/d1/notes/n1 {"content":"bye"}`,
		"DELETE /api/v1/devices/d1/notes/n1 ",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
