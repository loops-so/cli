package cmd

import (
	"net/http"
	"testing"
)

func TestRunCampaignsGet(t *testing.T) {
	body := `{
		"success": true,
		"id": "cmp_abc123",
		"emailMessageId": "em_abc123",
		"name": "Spring Launch",
		"status": "Draft",
		"createdAt": "2026-04-01T10:00:00Z",
		"updatedAt": "2026-04-02T10:00:00Z"
	}`

	t.Run("returns the campaign", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		c, err := runCampaignsGet(cfg(t), "cmp_abc123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.ID != "cmp_abc123" {
			t.Errorf("ID = %q, want cmp_abc123", c.ID)
		}
		if deref(c.EmailMessageID) != "em_abc123" {
			t.Errorf("EmailMessageID = %q, want em_abc123", deref(c.EmailMessageID))
		}
		if c.Name != "Spring Launch" {
			t.Errorf("Name = %q, want Spring Launch", c.Name)
		}
		if c.Status != "Draft" {
			t.Errorf("Status = %q, want Draft", c.Status)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Campaign not found"}`)
		_, err := runCampaignsGet(cfg(t), "cmp_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
