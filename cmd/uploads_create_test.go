package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// pngBytes is a minimal valid PNG header — enough for http.DetectContentType
// to return "image/png".
var pngBytes = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
}

type uploadServer struct {
	srv             *httptest.Server
	createReq       map[string]any
	putBody         []byte
	putContentType  string
	putContentLen   int64
	completeCalled  bool
	completeAssetID string
}

func serveUpload(t *testing.T) *uploadServer {
	t.Helper()
	keyring.MockInit()
	t.Setenv("LOOPS_CONFIG_DIR", t.TempDir())

	state := &uploadServer{}
	state.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/uploads":
			if err := json.NewDecoder(r.Body).Decode(&state.createReq); err != nil {
				t.Errorf("decode create body: %v", err)
			}
			fmt.Fprintf(w, `{"emailAssetId":"asset_123","presignedUrl":"%s/upload-target","expiresAt":"2026-05-21T12:00:00Z"}`, state.srv.URL)

		case r.Method == http.MethodPut && r.URL.Path == "/upload-target":
			state.putContentType = r.Header.Get("Content-Type")
			state.putContentLen = r.ContentLength
			state.putBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/uploads/") && strings.HasSuffix(r.URL.Path, "/complete"):
			state.completeCalled = true
			state.completeAssetID = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/uploads/"), "/complete")
			w.Write([]byte(`{"emailAssetId":"asset_123","finalUrl":"https://cdn.loops.so/asset_123.png"}`))

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(state.srv.Close)
	t.Setenv("LOOPS_API_KEY", "test-key")
	t.Setenv("LOOPS_ENDPOINT_URL", state.srv.URL)
	return state
}

func writeTempPNG(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	if err := os.WriteFile(path, pngBytes, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestRunUploadsCreate(t *testing.T) {
	t.Run("sniffs content-type when blank and uploads file body", func(t *testing.T) {
		state := serveUpload(t)
		path := writeTempPNG(t)

		result, err := runUploadsCreate(cfg(t), path, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.EmailAssetID != "asset_123" {
			t.Errorf("EmailAssetID = %q, want asset_123", result.EmailAssetID)
		}
		if result.FinalURL != "https://cdn.loops.so/asset_123.png" {
			t.Errorf("FinalURL = %q", result.FinalURL)
		}

		if state.createReq["contentType"] != "image/png" {
			t.Errorf("create contentType = %v, want image/png (sniffed)", state.createReq["contentType"])
		}
		if got, want := int64(state.createReq["contentLength"].(float64)), int64(len(pngBytes)); got != want {
			t.Errorf("create contentLength = %d, want %d", got, want)
		}

		if state.putContentType != "image/png" {
			t.Errorf("PUT Content-Type = %q, want image/png", state.putContentType)
		}
		if state.putContentLen != int64(len(pngBytes)) {
			t.Errorf("PUT Content-Length = %d, want %d", state.putContentLen, len(pngBytes))
		}
		if string(state.putBody) != string(pngBytes) {
			t.Errorf("PUT body did not match source file")
		}

		if !state.completeCalled {
			t.Error("complete endpoint was not called")
		}
		if state.completeAssetID != "asset_123" {
			t.Errorf("complete called for assetID %q, want asset_123", state.completeAssetID)
		}
	})

	t.Run("explicit --content-type overrides sniffing", func(t *testing.T) {
		state := serveUpload(t)
		path := writeTempPNG(t)

		if _, err := runUploadsCreate(cfg(t), path, "application/octet-stream"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if state.createReq["contentType"] != "application/octet-stream" {
			t.Errorf("create contentType = %v, want application/octet-stream", state.createReq["contentType"])
		}
		if state.putContentType != "application/octet-stream" {
			t.Errorf("PUT Content-Type = %q, want application/octet-stream", state.putContentType)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		serveUpload(t)
		if _, err := runUploadsCreate(cfg(t), "/does/not/exist.png", ""); err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("directory path returns error", func(t *testing.T) {
		serveUpload(t)
		if _, err := runUploadsCreate(cfg(t), t.TempDir(), ""); err == nil {
			t.Fatal("expected error for directory path, got nil")
		}
	})

	t.Run("create failure surfaces error", func(t *testing.T) {
		keyring.MockInit()
		t.Setenv("LOOPS_CONFIG_DIR", t.TempDir())
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"success":false,"message":"bad request"}`))
		}))
		t.Cleanup(srv.Close)
		t.Setenv("LOOPS_API_KEY", "test-key")
		t.Setenv("LOOPS_ENDPOINT_URL", srv.URL)

		if _, err := runUploadsCreate(cfg(t), writeTempPNG(t), ""); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
