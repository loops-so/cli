package cmd

import (
	"net/http"
	"testing"
)

func TestRunTransactionalDraft(t *testing.T) {
	body := `{
		"id": "tx_abc",
		"name": "Welcome",
		"draftEmailMessageId": "em_draft",
		"publishedEmailMessageId": "em_pub",
		"createdAt": "2026-01-01T00:00:00Z",
		"updatedAt": "2026-01-02T00:00:00Z",
		"dataVariables": [],
		"draftEmailMessageContentRevisionId": "rev_5"
	}`

	t.Run("returns draft", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		tx, err := runTransactionalDraft(cfg(t), "tx_abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deref(tx.DraftEmailMessageID) != "em_draft" {
			t.Errorf("DraftEmailMessageID = %q, want em_draft", deref(tx.DraftEmailMessageID))
		}
		if deref(tx.DraftEmailMessageContentRevisionID) != "rev_5" {
			t.Errorf("DraftEmailMessageContentRevisionID = %q, want rev_5", deref(tx.DraftEmailMessageContentRevisionID))
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"error":"not found"}`)
		_, err := runTransactionalDraft(cfg(t), "tx_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
