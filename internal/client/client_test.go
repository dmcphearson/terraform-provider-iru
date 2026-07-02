package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"
	"testing"
)

func newTestClient(t *testing.T, h http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := New(srv.URL, "test-token", "test")
	return c
}

func TestDoJSON_SendsBearerAndDecodes(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want Bearer test-token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"abc","name":"n"}`))
	}))

	var out Tag
	if err := c.DoJSON(context.Background(), "GET", "/api/v1/tags/abc", nil, &out); err != nil {
		t.Fatalf("DoJSON: %v", err)
	}
	if out.ID != "abc" || out.Name != "n" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestDoJSON_404ReturnsNotFound(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not found"}`))
	}))
	err := c.DoJSON(context.Background(), "GET", "/x", nil, nil)
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestDoJSON_RetriesOn503ThenSucceeds(t *testing.T) {
	orig := baseRetryDelay
	baseRetryDelay = time.Millisecond
	t.Cleanup(func() { baseRetryDelay = orig })

	var calls int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	// Shrink backoff so the test is fast.
	var out Tag
	if err := c.DoJSON(context.Background(), "GET", "/x", nil, &out); err != nil {
		t.Fatalf("DoJSON: %v", err)
	}
	if out.ID != "ok" {
		t.Errorf("out=%+v", out)
	}
	if calls != 3 {
		t.Errorf("calls=%d want 3", calls)
	}
}

func TestListAll_Paginates(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")
		// Page 1: 300 items (full page -> keep going); Page 2: 2 items (short -> stop).
		if offset == "0" {
			w.Write([]byte(`{"count":302,"results":[` + repeatTag(300) + `]}`))
			return
		}
		w.Write([]byte(`{"count":302,"results":[{"id":"a","name":"a"},{"id":"b","name":"b"}]}`))
	}))
	tags, err := c.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 302 {
		t.Errorf("got %d tags, want 302", len(tags))
	}
}

func TestGetTagByID_NotFoundWhenAbsent(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"count":1,"results":[{"id":"a","name":"a"}]}`))
	}))
	_, err := c.GetTagByID(context.Background(), "missing")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound for absent tag, got %v", err)
	}
}

func repeatTag(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ","
		}
		s += `{"id":"x","name":"x"}`
	}
	return s
}
