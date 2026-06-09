package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunTransactionalUpdate(t *testing.T) {
	body := `{
		"id": "tx_abc",
		"name": "Renamed",
		"draftEmailMessageId": "em_draft",
		"publishedEmailMessageId": "em_pub",
		"createdAt": "2026-01-01T00:00:00Z",
		"updatedAt": "2026-01-03T00:00:00Z",
		"dataVariables": []
	}`

	t.Run("returns updated transactional", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		tx, err := runTransactionalUpdate(cfg(t), "tx_abc", loops.UpdateTransactionalRequest{Name: "Renamed"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.Name != "Renamed" {
			t.Errorf("Name = %q, want Renamed", tx.Name)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"error":"not found"}`)
		_, err := runTransactionalUpdate(cfg(t), "tx_missing", loops.UpdateTransactionalRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
