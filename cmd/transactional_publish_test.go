package cmd

import (
	"net/http"
	"testing"
)

func TestRunTransactionalPublish(t *testing.T) {
	body := `{
		"id": "tx_abc",
		"name": "Welcome",
		"draftEmailMessageId": null,
		"publishedEmailMessageId": "em_pub_new",
		"createdAt": "2026-01-01T00:00:00Z",
		"updatedAt": "2026-01-04T00:00:00Z",
		"dataVariables": []
	}`

	t.Run("returns published transactional", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		tx, err := runTransactionalPublish(cfg(t), "tx_abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deref(tx.PublishedEmailMessageID) != "em_pub_new" {
			t.Errorf("PublishedEmailMessageID = %q, want em_pub_new", deref(tx.PublishedEmailMessageID))
		}
		if tx.DraftEmailMessageID != nil {
			t.Errorf("DraftEmailMessageID = %v, want nil", tx.DraftEmailMessageID)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusConflict, `{"error":"no draft to publish"}`)
		_, err := runTransactionalPublish(cfg(t), "tx_abc")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
