package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

// UploadTarget is the presigned S3 POST returned by the custom-app and ipa-app
// upload endpoints.
type UploadTarget struct {
	PostURL  string            `json:"post_url"`
	PostData map[string]string `json:"post_data"`
	FileKey  string            `json:"file_key"`
}

// UploadFile streams the file at path to a presigned S3 POST. The policy fields go
// first and the file last, as S3 requires. The request carries no Iru credentials.
// S3 rejects chunked POSTs, so the body length is computed up front instead of
// buffering the whole file.
func (c *Client) UploadFile(ctx context.Context, t UploadTarget, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening upload file: %w", err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("reading upload file: %w", err)
	}

	var head bytes.Buffer
	mw := multipart.NewWriter(&head)
	keys := make([]string, 0, len(t.PostData))
	for k := range t.PostData {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := mw.WriteField(k, t.PostData[k]); err != nil {
			return fmt.Errorf("building upload form: %w", err)
		}
	}
	if _, err := mw.CreateFormFile("file", filepath.Base(path)); err != nil {
		return fmt.Errorf("building upload form: %w", err)
	}
	tail := "\r\n--" + mw.Boundary() + "--\r\n"

	body := io.MultiReader(bytes.NewReader(head.Bytes()), f, bytes.NewReader([]byte(tail)))
	req, err := http.NewRequestWithContext(ctx, "POST", t.PostURL, body)
	if err != nil {
		return fmt.Errorf("building upload request: %w", err)
	}
	req.ContentLength = int64(head.Len()) + info.Size() + int64(len(tail))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.uploadClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &APIError{StatusCode: resp.StatusCode, Body: string(b)}
	}
	return nil
}
