package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunTransactionalUpdate(t *testing.T) {
	body := `{
		"id": "tx_abc",
		"name": "Renamed",
		"transactionalGroupId": "tg_2",
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
		if deref(tx.TransactionalGroupID) != "tg_2" {
			t.Errorf("TransactionalGroupID = %q, want tg_2", deref(tx.TransactionalGroupID))
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"error":"not found"}`)
		_, err := runTransactionalUpdate(cfg(t), "tx_missing", loops.UpdateTransactionalRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("sends transactionalGroupId when provided", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, body)
		_, err := runTransactionalUpdate(cfg(t), "tx_abc", loops.UpdateTransactionalRequest{
			Name:                 "Renamed",
			TransactionalGroupID: "tg_2",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if sent["transactionalGroupId"] != "tg_2" {
			t.Errorf("transactionalGroupId = %v, want tg_2", sent["transactionalGroupId"])
		}
	})
}
