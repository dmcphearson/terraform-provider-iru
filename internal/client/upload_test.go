package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// S3 needs every policy field before the file, a fixed Content-Length, and no Iru
// bearer token on the request.
func TestUploadFile_PresignedPost(t *testing.T) {
	var gotOrder []string
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("upload must not send the Iru token to S3")
		}
		if r.ContentLength <= 0 || len(r.TransferEncoding) > 0 {
			t.Errorf("want fixed Content-Length, got %d %v", r.ContentLength, r.TransferEncoding)
		}
		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			gotOrder = append(gotOrder, part.FormName())
			if part.FormName() == "file" {
				b, _ := io.ReadAll(part)
				if string(b) != "ipa-bytes" || part.FileName() != "app.ipa" {
					t.Errorf("file part = %q %q", part.FileName(), b)
				}
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(s3.Close)

	c := New("http://unused", "secret", "test")
	err := c.UploadFile(context.Background(), UploadTarget{
		PostURL:  s3.URL,
		PostData: map[string]string{"key": "k", "policy": "p"},
	}, writeTempFile(t, "app.ipa", "ipa-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(gotOrder, ",") != "key,policy,file" {
		t.Errorf("part order = %v, want fields then file", gotOrder)
	}
}

func TestUploadInHouseApp_PollsUntilValidated(t *testing.T) {
	uploadPollInterval = time.Millisecond
	t.Cleanup(func() { uploadPollInterval = time.Second })

	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(s3.Close)

	polls := 0
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/library/ipa-apps/upload":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["filename"] != "app.ipa" {
				t.Errorf("filename = %q", body["filename"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "up-1", "post_url": s3.URL, "file_key": "tenants/x/app.ipa",
				"post_data": map[string]string{"key": "tenants/x/app.ipa"},
			})
		case "/api/v1/library/ipa-apps/upload/up-1/status":
			polls++
			status := "UPLOADING"
			if polls >= 2 {
				status = "VALIDATED"
			}
			_, _ = w.Write([]byte(`{"status":"` + status + `"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))

	key, err := c.UploadInHouseApp(context.Background(), writeTempFile(t, "app.ipa", "x"))
	if err != nil {
		t.Fatal(err)
	}
	if key != "tenants/x/app.ipa" || polls != 2 {
		t.Errorf("key = %q after %d polls", key, polls)
	}
}

func TestUploadInHouseApp_ValidationFailure(t *testing.T) {
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(s3.Close)
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/status") {
			_, _ = w.Write([]byte(`{"status":"VALIDATE_FAILED","reason":"not an ipa"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"up-1","post_url":"` + s3.URL + `","file_key":"k","post_data":{}}`))
	}))

	_, err := c.UploadInHouseApp(context.Background(), writeTempFile(t, "app.ipa", "x"))
	if err == nil || !strings.Contains(err.Error(), "VALIDATE_FAILED") {
		t.Fatalf("err = %v, want VALIDATE_FAILED", err)
	}
}

// Create documents no response body; the new item is then found by file_key.
func TestCreateInHouseApp_FindsByFileKeyWhenBodyEmpty(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			return
		}
		_, _ = w.Write([]byte(`{"count":2,"results":[{"id":"a","file_key":"other"},{"id":"b","file_key":"mine"}]}`))
	}))

	out, err := c.CreateInHouseApp(context.Background(), InHouseApp{Name: "n", FileKey: "mine"})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "b" {
		t.Errorf("id = %q, want b", out.ID)
	}
}

func TestUploadCustomApp(t *testing.T) {
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(s3.Close)
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/library/custom-apps/upload" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "tool.pkg" {
			t.Errorf("name = %q", body["name"])
		}
		_, _ = w.Write([]byte(`{"post_url":"` + s3.URL + `","file_key":"library/custom_apps/tool.pkg","post_data":{"key":"k"}}`))
	}))

	key, err := c.UploadCustomApp(context.Background(), writeTempFile(t, "tool.pkg", "pkg"))
	if err != nil {
		t.Fatal(err)
	}
	if key != "library/custom_apps/tool.pkg" {
		t.Errorf("key = %q", key)
	}
}
