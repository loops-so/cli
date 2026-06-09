package cmd

import (
	"net/http"
	"testing"
)

func TestRunTransactionalGet(t *testing.T) {
	body := `{
		"id": "tx_abc",
		"name": "Welcome",
		"draftEmailMessageId": "em_draft",
		"publishedEmailMessageId": "em_pub",
		"createdAt": "2026-01-01T00:00:00Z",
		"updatedAt": "2026-01-02T00:00:00Z",
		"dataVariables": ["name"]
	}`

	t.Run("returns the transactional", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		tx, err := runTransactionalGet(cfg(t), "tx_abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.ID != "tx_abc" {
			t.Errorf("ID = %q, want tx_abc", tx.ID)
		}
		if tx.Name != "Welcome" {
			t.Errorf("Name = %q, want Welcome", tx.Name)
		}
		if deref(tx.DraftEmailMessageID) != "em_draft" {
			t.Errorf("DraftEmailMessageID = %q, want em_draft", deref(tx.DraftEmailMessageID))
		}
		if deref(tx.PublishedEmailMessageID) != "em_pub" {
			t.Errorf("PublishedEmailMessageID = %q, want em_pub", deref(tx.PublishedEmailMessageID))
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"error":"not found"}`)
		_, err := runTransactionalGet(cfg(t), "tx_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
