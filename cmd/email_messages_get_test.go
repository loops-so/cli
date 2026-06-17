package cmd

import (
	"net/http"
	"testing"
)

func TestRunEmailMessagesGet(t *testing.T) {
	body := `{
		"success": true,
		"id": "em_abc123",
		"campaignId": "cmp_xyz789",
		"subject": "Hello",
		"previewText": "Preview",
		"fromName": "Acme",
		"fromEmail": "hello",
		"replyToEmail": "support@acme.com",
		"lmx": "<Paragraph>Hi</Paragraph>",
		"contentRevisionId": "rev_1",
		"updatedAt": "2026-04-20T10:00:00Z"
	}`

	t.Run("returns the email message", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		msg, err := runEmailMessagesGet(cfg(t), "em_abc123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.ID != "em_abc123" {
			t.Errorf("ID = %q, want em_abc123", msg.ID)
		}
		if deref(msg.CampaignID) != "cmp_xyz789" {
			t.Errorf("CampaignID = %q, want cmp_xyz789", deref(msg.CampaignID))
		}
		if msg.Subject != "Hello" {
			t.Errorf("Subject = %q, want Hello", msg.Subject)
		}
		if deref(msg.ContentRevisionID) != "rev_1" {
			t.Errorf("ContentRevisionID = %q, want rev_1", deref(msg.ContentRevisionID))
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Email message not found"}`)
		_, err := runEmailMessagesGet(cfg(t), "em_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
