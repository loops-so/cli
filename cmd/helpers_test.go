package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loops-so/cli/internal/config"
	"github.com/zalando/go-keyring"
)

func cfg(t *testing.T) *config.Config {
	t.Helper()
	c, err := config.Load("")
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return c
}

func mockKeyring(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	t.Setenv("LOOPS_CONFIG_DIR", t.TempDir())
}

func serveJSON(t *testing.T, status int, body string) {
	t.Helper()
	keyring.MockInit()
	t.Setenv("LOOPS_CONFIG_DIR", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("LOOPS_API_KEY", "test-key")
	t.Setenv("LOOPS_ENDPOINT_URL", srv.URL)
}

// capturedRequest records the most recent HTTP request made by the SDK during
// a test. Tests that need to assert on the request body, method, or path use
// serveJSONCapture and inspect the returned pointer after the call.
type capturedRequest struct {
	Method string
	Path   string
	Body   []byte
}

func serveJSONCapture(t *testing.T, status int, body string) *capturedRequest {
	t.Helper()
	keyring.MockInit()
	t.Setenv("LOOPS_CONFIG_DIR", t.TempDir())
	cap := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.Method = r.Method
		cap.Path = r.URL.RequestURI()
		cap.Body, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("LOOPS_API_KEY", "test-key")
	t.Setenv("LOOPS_ENDPOINT_URL", srv.URL)
	return cap
}
