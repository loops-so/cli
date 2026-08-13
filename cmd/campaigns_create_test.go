package cmd

import (
	"encoding/json"
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

	t.Run("sends targeting and scheduling fields", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, body)
		mailingList := "ml_1"
		ts := "2026-07-01T12:00:00Z"
		_, err := runCampaignsCreate(cfg(t), loops.CreateCampaignRequest{
			Name:            "Spring",
			CampaignGroupID: "cg_1",
			MailingListID:   &mailingList,
			Scheduling: &loops.CampaignSchedulingRequest{
				Method:    loops.CampaignSchedulingMethodSchedule,
				Timestamp: ts,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		if sent["name"] != "Spring" {
			t.Errorf("name = %v, want Spring", sent["name"])
		}
		if sent["campaignGroupId"] != "cg_1" {
			t.Errorf("campaignGroupId = %v, want cg_1", sent["campaignGroupId"])
		}
		if sent["mailingListId"] != "ml_1" {
			t.Errorf("mailingListId = %v, want ml_1", sent["mailingListId"])
		}
		sched, ok := sent["scheduling"].(map[string]any)
		if !ok {
			t.Fatalf("scheduling not an object: %v", sent["scheduling"])
		}
		if sched["method"] != "schedule" || sched["timestamp"] != ts {
			t.Errorf("scheduling = %v", sched)
		}
	})

	t.Run("sends audience segment and filter together", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, body)
		segment := "seg_1"
		_, err := runCampaignsCreate(cfg(t), loops.CreateCampaignRequest{
			Name:              "Spring",
			AudienceSegmentID: &segment,
			AudienceFilter: &loops.AudienceFilter{
				Match: "all",
				Conditions: []loops.AudienceFilterCondition{
					{
						Type:     loops.AudienceConditionTypeProperty,
						Property: &loops.PropertyCondition{Key: "plan", Operator: "equals"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		if sent["audienceSegmentId"] != "seg_1" {
			t.Errorf("audienceSegmentId = %v, want seg_1", sent["audienceSegmentId"])
		}
		filter, ok := sent["audienceFilter"].(map[string]any)
		if !ok {
			t.Fatalf("audienceFilter not an object: %v", sent["audienceFilter"])
		}
		if filter["match"] != "all" {
			t.Errorf("audienceFilter.match = %v, want all", filter["match"])
		}
	})
}
