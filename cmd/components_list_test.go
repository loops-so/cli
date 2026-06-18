package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunComponentsList(t *testing.T) {
	t.Run("returns components", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"pagination":{"nextCursor":""},"data":[{"id":"cmpt_1","name":"Header","lmx":"<H1>Hello</H1>"}]}`)
		components, err := runComponentsList(cfg(t), loops.PaginationParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(components) != 1 {
			t.Fatalf("expected 1 component, got %d", len(components))
		}
		if components[0].ID != "cmpt_1" {
			t.Errorf("ID = %q, want cmpt_1", components[0].ID)
		}
		if components[0].Name != "Header" {
			t.Errorf("Name = %q, want Header", components[0].Name)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"error":"unauthorized"}`)
		_, err := runComponentsList(cfg(t), loops.PaginationParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
