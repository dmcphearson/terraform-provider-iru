package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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
