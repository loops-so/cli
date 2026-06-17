package cmd

import (
	"net/http"
	"testing"
)

func TestRunComponentsGet(t *testing.T) {
	body := `{
		"success": true,
		"id": "cmpt_abc123",
		"name": "Header",
		"lmx": "<H1>Hello</H1>"
	}`

	t.Run("returns the component", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		c, err := runComponentsGet(cfg(t), "cmpt_abc123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.ID != "cmpt_abc123" {
			t.Errorf("ID = %q, want cmpt_abc123", c.ID)
		}
		if c.Name != "Header" {
			t.Errorf("Name = %q, want Header", c.Name)
		}
		if c.LMX != "<H1>Hello</H1>" {
			t.Errorf("LMX = %q, want <H1>Hello</H1>", c.LMX)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Component not found"}`)
		_, err := runComponentsGet(cfg(t), "cmpt_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
