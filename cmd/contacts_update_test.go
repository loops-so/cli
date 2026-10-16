package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunContactsUpdate(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"success":true,"id":"cnt_abc123"}`)
		err := runContactsUpdate(cfg(t), loops.UpdateContactRequest{
			Email:     "bob@example.com",
			FirstName: "Bob",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("sends source", func(t *testing.T) {
		captured := serveJSONCapture(t, http.StatusOK, `{"success":true,"id":"cnt_abc123"}`)
		err := runContactsUpdate(cfg(t), loops.UpdateContactRequest{
			Email:  "bob@example.com",
			Source: "cli",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var body map[string]any
		if err := json.Unmarshal(captured.Body, &body); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if body["source"] != "cli" {
			t.Errorf("body[\"source\"] = %v, want %q", body["source"], "cli")
		}
	})

	t.Run("returns error on API failure", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"success":false,"message":"Invalid email address"}`)
		err := runContactsUpdate(cfg(t), loops.UpdateContactRequest{Email: "notanemail"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
