package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunCampaignsCreate(t *testing.T) {
	body := `{
		"success": true,
		"id": "cmp_new",
		"name": "Spring",
		"status": "Draft",
		"createdAt": "2026-04-20T10:00:00Z",
		"updatedAt": "2026-04-20T10:00:00Z",
		"emailMessageId": "em_new",
		"emailMessageContentRevisionId": "rev_1"
	}`

	t.Run("returns response on success", func(t *testing.T) {
		serveJSON(t, http.StatusCreated, body)
		resp, err := runCampaignsCreate(cfg(t), loops.CreateCampaignRequest{Name: "Spring"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != "cmp_new" {
			t.Errorf("ID = %q, want cmp_new", resp.ID)
		}
		if deref(resp.EmailMessageID) != "em_new" {
			t.Errorf("EmailMessageID = %q, want em_new", deref(resp.EmailMessageID))
		}
		if deref(resp.EmailMessageContentRevisionID) != "rev_1" {
			t.Errorf("EmailMessageContentRevisionID = %q, want rev_1", deref(resp.EmailMessageContentRevisionID))
		}
	})

	t.Run("returns error on non-201 response", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"success":false,"message":"name is required"}`)
		_, err := runCampaignsCreate(cfg(t), loops.CreateCampaignRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
