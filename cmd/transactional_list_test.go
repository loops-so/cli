package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunTransactionalList(t *testing.T) {
	t.Run("returns emails", func(t *testing.T) {
		body := `{
			"pagination": {"nextCursor": ""},
			"data": [{
				"id": "tx_1",
				"name": "Welcome",
				"draftEmailMessageId": "em_draft_1",
				"publishedEmailMessageId": "em_pub_1",
				"createdAt": "2026-01-01T00:00:00Z",
				"updatedAt": "2026-01-02T00:00:00Z",
				"dataVariables": ["name", "company"]
			}]
		}`
		serveJSON(t, http.StatusOK, body)
		emails, err := runTransactionalList(cfg(t), loops.PaginationParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(emails) != 1 {
			t.Fatalf("len(emails) = %d, want 1", len(emails))
		}
		e := emails[0]
		if e.ID != "tx_1" {
			t.Errorf("ID = %q, want tx_1", e.ID)
		}
		if e.Name != "Welcome" {
			t.Errorf("Name = %q, want Welcome", e.Name)
		}
		if deref(e.DraftEmailMessageID) != "em_draft_1" {
			t.Errorf("DraftEmailMessageID = %q, want em_draft_1", deref(e.DraftEmailMessageID))
		}
		if deref(e.PublishedEmailMessageID) != "em_pub_1" {
			t.Errorf("PublishedEmailMessageID = %q, want em_pub_1", deref(e.PublishedEmailMessageID))
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"error":"unauthorized"}`)
		_, err := runTransactionalList(cfg(t), loops.PaginationParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
