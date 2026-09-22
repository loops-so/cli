package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/loops-so/cli/internal/cmdutil"
	"github.com/loops-so/loops-go"
	"github.com/zalando/go-keyring"
)

func TestParseMailingLists(t *testing.T) {
	t.Run("valid true and false", func(t *testing.T) {
		m, err := parseMailingLists([]string{"abc=true", "def=false"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m["abc"] != true {
			t.Errorf("expected abc=true, got %v", m["abc"])
		}
		if m["def"] != false {
			t.Errorf("expected def=false, got %v", m["def"])
		}
	})

	t.Run("case-insensitive", func(t *testing.T) {
		m, err := parseMailingLists([]string{"abc=True", "def=FALSE"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m["abc"] != true {
			t.Errorf("expected abc=true, got %v", m["abc"])
		}
		if m["def"] != false {
			t.Errorf("expected def=false, got %v", m["def"])
		}
	})

	t.Run("missing equals", func(t *testing.T) {
		_, err := parseMailingLists([]string{"badvalue"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "expected id=true|false") {
			t.Errorf("error %q should mention expected id=true|false", err.Error())
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		_, err := parseMailingLists([]string{"abc=maybe"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), `value must be "true" or "false"`) {
			t.Errorf("error %q should mention value must be true or false", err.Error())
		}
	})

	t.Run("empty returns nil", func(t *testing.T) {
		m, err := parseMailingLists(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m != nil {
			t.Errorf("expected nil map, got %v", m)
		}
	})
}

func serveEventsSend(t *testing.T, status int, body string, check func(*http.Request)) {
	t.Helper()
	keyring.MockInit()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("LOOPS_API_KEY", "test-key")
	t.Setenv("LOOPS_ENDPOINT_URL", srv.URL)
}

func TestEventsSend(t *testing.T) {
	t.Run("happy path with email", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"success":true}`)
		err := runEventsSend(cfg(t), loops.SendEventRequest{
			Email:     "user@example.com",
			EventName: "user-signed-up",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("happy path with userId", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"success":true}`)
		err := runEventsSend(cfg(t), loops.SendEventRequest{
			UserID:    "user-123",
			EventName: "user-signed-up",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("api error", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"error":"invalid event"}`)
		err := runEventsSend(cfg(t), loops.SendEventRequest{
			Email:     "user@example.com",
			EventName: "bad-event",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid event") {
			t.Errorf("error %q should mention invalid event", err.Error())
		}
	})

	t.Run("missing both email and userId returns validation error", func(t *testing.T) {
		email := ""
		userID := ""
		if email == "" && userID == "" {
			err := fmt.Errorf("at least one of --email or --user-id is required")
			if !strings.Contains(err.Error(), "--email") {
				t.Errorf("error should mention --email")
			}
		}
	})

	t.Run("props valid JSON", func(t *testing.T) {
		var captured map[string]any
		serveEventsSend(t, http.StatusOK, `{"success":true}`, func(r *http.Request) {
			json.NewDecoder(r.Body).Decode(&captured)
		})
		f, _ := os.CreateTemp(t.TempDir(), "props-*.json")
		json.NewEncoder(f).Encode(map[string]any{"plan": "pro", "trial": true})
		f.Close()

		props, err := cmdutil.ParseJSONFile("props", f.Name())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if props["plan"] != "pro" {
			t.Errorf("expected plan=pro, got %v", props["plan"])
		}
	})

	t.Run("props file not found", func(t *testing.T) {
		_, err := cmdutil.ParseJSONFile("props", "/nonexistent/props.json")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "--props") {
			t.Errorf("error %q should mention --props", err.Error())
		}
	})

	t.Run("props invalid JSON", func(t *testing.T) {
		f, _ := os.CreateTemp(t.TempDir(), "props-*.json")
		f.WriteString("not json")
		f.Close()

		_, err := cmdutil.ParseJSONFile("props", f.Name())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "--props must be a valid JSON object") {
			t.Errorf("error %q should mention --props must be a valid JSON object", err.Error())
		}
	})

	t.Run("contact-props merged at top level", func(t *testing.T) {
		var captured map[string]any
		serveEventsSend(t, http.StatusOK, `{"success":true}`, func(r *http.Request) {
			json.NewDecoder(r.Body).Decode(&captured)
		})

		f, _ := os.CreateTemp(t.TempDir(), "contact-*.json")
		json.NewEncoder(f).Encode(map[string]any{"firstName": "Alice", "age": float64(30)})
		f.Close()

		contactProps, err := cmdutil.ParseJSONFile("contact-props", f.Name())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = runEventsSend(cfg(t), loops.SendEventRequest{
			Email:             "user@example.com",
			EventName:         "upgrade",
			ContactProperties: contactProps,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if captured["firstName"] != "Alice" {
			t.Errorf("expected firstName=Alice in body, got %v", captured["firstName"])
		}
	})

	t.Run("list single and multiple", func(t *testing.T) {
		m, err := parseMailingLists([]string{"list1=true", "list2=false"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m["list1"] != true || m["list2"] != false {
			t.Errorf("unexpected map: %v", m)
		}
	})

	t.Run("list invalid value", func(t *testing.T) {
		_, err := parseMailingLists([]string{"list1=maybe"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("idempotency-key forwarded as header", func(t *testing.T) {
		var capturedKey string
		serveEventsSend(t, http.StatusOK, `{"success":true}`, func(r *http.Request) {
			capturedKey = r.Header.Get("Idempotency-Key")
		})
		err := runEventsSend(cfg(t), loops.SendEventRequest{
			Email:          "user@example.com",
			EventName:      "test",
			IdempotencyKey: "my-key-123",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedKey != "my-key-123" {
			t.Errorf("expected Idempotency-Key=my-key-123, got %q", capturedKey)
		}
	})

	t.Run("json output mode", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"success":true}`)
		var buf bytes.Buffer
		t.Setenv("OUTPUT_FORMAT", "json")

		err := runEventsSend(cfg(t), loops.SendEventRequest{
			Email:     "user@example.com",
			EventName: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = buf
	})

	t.Run("props with eventProperties wrapper", func(t *testing.T) {
		f, _ := os.CreateTemp(t.TempDir(), "props-*.json")
		json.NewEncoder(f).Encode(map[string]any{
			"eventProperties": map[string]any{"plan": "pro", "trial": true},
		})
		f.Close()

		props, err := cmdutil.ParseJSONFile("props", f.Name())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var eventProps map[string]any
		if nested, ok := props["eventProperties"]; ok {
			if m, ok := nested.(map[string]any); ok {
				eventProps = m
			}
		} else {
			eventProps = props
		}

		if eventProps["plan"] != "pro" {
			t.Errorf("expected plan=pro, got %v", eventProps["plan"])
		}
	})
}
