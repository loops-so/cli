package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunTransactionalCreate(t *testing.T) {
	body := `{
		"id": "tx_new",
		"name": "Welcome",
		"transactionalGroupId": "tg_1",
		"draftEmailMessageId": "em_draft",
		"publishedEmailMessageId": null,
		"createdAt": "2026-01-01T00:00:00Z",
		"updatedAt": "2026-01-01T00:00:00Z",
		"dataVariables": [],
		"draftEmailMessageContentRevisionId": "rev_1"
	}`

	t.Run("returns response on success", func(t *testing.T) {
		serveJSON(t, http.StatusCreated, body)
		tx, err := runTransactionalCreate(cfg(t), loops.CreateTransactionalRequest{Name: "Welcome"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.ID != "tx_new" {
			t.Errorf("ID = %q, want tx_new", tx.ID)
		}
		if deref(tx.TransactionalGroupID) != "tg_1" {
			t.Errorf("TransactionalGroupID = %q, want tg_1", deref(tx.TransactionalGroupID))
		}
		if deref(tx.DraftEmailMessageID) != "em_draft" {
			t.Errorf("DraftEmailMessageID = %q, want em_draft", deref(tx.DraftEmailMessageID))
		}
		if deref(tx.DraftEmailMessageContentRevisionID) != "rev_1" {
			t.Errorf("DraftEmailMessageContentRevisionID = %q, want rev_1", deref(tx.DraftEmailMessageContentRevisionID))
		}
	})

	t.Run("returns error on non-201 response", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"error":"name is required"}`)
		_, err := runTransactionalCreate(cfg(t), loops.CreateTransactionalRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("sends transactionalGroupId when provided", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, body)
		_, err := runTransactionalCreate(cfg(t), loops.CreateTransactionalRequest{
			Name:                 "Welcome",
			TransactionalGroupID: "tg_1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if sent["transactionalGroupId"] != "tg_1" {
			t.Errorf("transactionalGroupId = %v, want tg_1", sent["transactionalGroupId"])
		}
	})

	t.Run("omits transactionalGroupId when empty", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, body)
		_, err := runTransactionalCreate(cfg(t), loops.CreateTransactionalRequest{Name: "Welcome"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := sent["transactionalGroupId"]; ok {
			t.Errorf("transactionalGroupId should be omitted when empty, got %v", sent)
		}
	})
}
