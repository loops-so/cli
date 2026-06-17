package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunCampaignsUpdate(t *testing.T) {
	body := `{
		"success": true,
		"id": "cmp_abc123",
		"emailMessageId": "em_abc123",
		"name": "Renamed",
		"status": "Draft",
		"createdAt": "2026-04-01T10:00:00Z",
		"updatedAt": "2026-04-25T10:00:00Z"
	}`

	t.Run("returns campaign on success", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		c, err := runCampaignsUpdate(cfg(t), "cmp_abc123", loops.UpdateCampaignRequest{Name: "Renamed"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.ID != "cmp_abc123" {
			t.Errorf("ID = %q, want cmp_abc123", c.ID)
		}
		if c.Name != "Renamed" {
			t.Errorf("Name = %q, want Renamed", c.Name)
		}
		if deref(c.EmailMessageID) != "em_abc123" {
			t.Errorf("EmailMessageID = %q, want em_abc123", deref(c.EmailMessageID))
		}
	})

	t.Run("returns error when not in draft", func(t *testing.T) {
		serveJSON(t, http.StatusConflict, `{"success":false,"message":"Campaign is not in draft status"}`)
		_, err := runCampaignsUpdate(cfg(t), "cmp_abc123", loops.UpdateCampaignRequest{Name: "Renamed"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("nil pointer + Set sends JSON null on the wire", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, body)
		_, err := runCampaignsUpdate(cfg(t), "cmp_abc123", loops.UpdateCampaignRequest{
			MailingListID: nil,
			Set:           map[string]bool{"mailingListId": true},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		v, ok := sent["mailingListId"]
		if !ok {
			t.Fatalf("mailingListId missing from request body: %v", sent)
		}
		if v != nil {
			t.Errorf("mailingListId = %v, want nil (JSON null)", v)
		}
	})

	t.Run("Set map controls which fields are sent", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, body)
		ts := "2026-07-01T12:00:00Z"
		_, err := runCampaignsUpdate(cfg(t), "cmp_abc123", loops.UpdateCampaignRequest{
			Name: "Renamed",
			Scheduling: &loops.CampaignSchedulingRequest{
				Method:    loops.CampaignSchedulingMethodSchedule,
				Timestamp: ts,
			},
			Set: map[string]bool{
				"name":       true,
				"scheduling": true,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		if _, ok := sent["campaignGroupId"]; ok {
			t.Errorf("unset campaignGroupId leaked into request: %v", sent)
		}
		if _, ok := sent["mailingListId"]; ok {
			t.Errorf("unset mailingListId leaked into request: %v", sent)
		}
		if sent["name"] != "Renamed" {
			t.Errorf("name = %v, want Renamed", sent["name"])
		}
		sched, ok := sent["scheduling"].(map[string]any)
		if !ok {
			t.Fatalf("scheduling not an object: %v", sent["scheduling"])
		}
		if sched["method"] != "schedule" || sched["timestamp"] != ts {
			t.Errorf("scheduling = %v", sched)
		}
	})
}
