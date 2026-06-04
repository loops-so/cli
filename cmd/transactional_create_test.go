package cmd

import (
	"net/http"
	"testing"
)

func TestRunTransactionalCreate(t *testing.T) {
	body := `{
		"id": "tx_new",
		"name": "Usage alert",
		"draftEmailMessageId": "em_new",
		"draftEmailMessageContentRevisionId": "rev_1",
		"publishedEmailMessageId": null,
		"createdAt": "2026-06-04T10:00:00Z",
		"updatedAt": "2026-06-04T10:00:00Z",
		"dataVariables": []
	}`

	t.Run("returns response on success", func(t *testing.T) {
		serveJSON(t, http.StatusCreated, body)
		resp, err := runTransactionalCreate(cfg(t), "Usage alert")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != "tx_new" {
			t.Errorf("ID = %q, want tx_new", resp.ID)
		}
		if resp.DraftEmailMessageID != "em_new" {
			t.Errorf("DraftEmailMessageID = %q, want em_new", resp.DraftEmailMessageID)
		}
		if resp.DraftEmailMessageContentRevisionID != "rev_1" {
			t.Errorf("DraftEmailMessageContentRevisionID = %q, want rev_1", resp.DraftEmailMessageContentRevisionID)
		}
	})

	t.Run("returns error on non-201 response", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"message":"name is required"}`)
		_, err := runTransactionalCreate(cfg(t), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
