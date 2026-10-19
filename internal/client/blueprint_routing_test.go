package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
)

// PATCH sends only the enrollment_code keys asked for.
func TestUpdateBlueprintRouting_SendsOnlyGivenKeys(t *testing.T) {
	var method, path, body string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		method, path, body = r.Method, r.URL.Path, string(b)
		_, _ = w.Write([]byte(`{"enrollment_code":{"code":"123456","is_active":false}}`))
	}))
	out, err := c.UpdateBlueprintRouting(context.Background(), map[string]any{"is_active": false})
	if err != nil {
		t.Fatal(err)
	}
	if method != "PATCH" || path != "/api/v1/blueprint-routing/" || body != `{"enrollment_code":{"is_active":false}}` {
		t.Errorf("request = %s %s %s", method, path, body)
	}
	if out.EnrollmentCode.Code != "123456" {
		t.Errorf("code = %q", out.EnrollmentCode.Code)
	}
}

// Activity stops at max even when more pages exist, and keeps per-type details raw.
func TestListBlueprintRoutingActivity_StopsAtMax(t *testing.T) {
	var calls int
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		_, _ = fmt.Fprint(w, `{"count":1000,"next":"more","results":[`)
		for i := 0; i < limit; i++ {
			if i > 0 {
				_, _ = fmt.Fprint(w, ",")
			}
			_, _ = fmt.Fprintf(w, `{"id":%d,"activity_type":"enrollment","user":null,"device_id":"d1","details":{"enrollment_type":"DEP"}}`, offset+i)
		}
		_, _ = fmt.Fprint(w, `]}`)
	}))
	got, err := c.ListBlueprintRoutingActivity(context.Background(), 350)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 350 || calls != 2 {
		t.Fatalf("got %d events in %d calls, want 350 in 2", len(got), calls)
	}
	if got[0].User != nil || got[0].DeviceID == nil || string(got[0].Details) != `{"enrollment_type":"DEP"}` {
		t.Errorf("event = %+v", got[0])
	}
}
