package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunCampaignsList(t *testing.T) {
	t.Run("returns campaigns", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"pagination":{"nextCursor":""},"data":[{"id":"cmp_1","emailMessageId":"em_1","name":"Spring","status":"Draft","createdAt":"2026-04-01","updatedAt":"2026-04-02","scheduling":{"method":"now","timestamp":null}}]}`)
		campaigns, err := runCampaignsList(cfg(t), loops.ListCampaignsParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(campaigns) != 1 {
			t.Fatalf("expected 1 campaign, got %d", len(campaigns))
		}
		if campaigns[0].ID != "cmp_1" {
			t.Errorf("ID = %q, want cmp_1", campaigns[0].ID)
		}
		if deref(campaigns[0].EmailMessageID) != "em_1" {
			t.Errorf("EmailMessageID = %q, want em_1", deref(campaigns[0].EmailMessageID))
		}
	})

	t.Run("sends group filter", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, `{"pagination":{"nextCursor":""},"data":[]}`)
		if _, err := runCampaignsList(cfg(t), loops.ListCampaignsParams{CampaignGroupID: "cg_1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cap.Path != "/campaigns?campaignGroupId=cg_1" {
			t.Errorf("Path = %q, want /campaigns?campaignGroupId=cg_1", cap.Path)
		}
	})

	t.Run("sends group filter with cursor", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, `{"pagination":{"nextCursor":""},"data":[]}`)
		_, err := runCampaignsList(cfg(t), loops.ListCampaignsParams{
			PaginationParams: loops.PaginationParams{Cursor: "abc"},
			CampaignGroupID:  "cg_1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cap.Path != "/campaigns?campaignGroupId=cg_1&cursor=abc" {
			t.Errorf("Path = %q, want /campaigns?campaignGroupId=cg_1&cursor=abc", cap.Path)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"error":"unauthorized"}`)
		_, err := runCampaignsList(cfg(t), loops.ListCampaignsParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
