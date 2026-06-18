package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunAudienceSegmentsGet(t *testing.T) {
	t.Run("returns segment with nontrivial filter", func(t *testing.T) {
		body := `{
			"id": "seg_abc",
			"name": "Active pro users",
			"description": "Pro plan + opted in + active",
			"createdAt": "2026-04-01T10:00:00Z",
			"updatedAt": "2026-04-20T10:00:00Z",
			"filter": {
				"match": "all",
				"conditions": [
					{"type": "property", "key": "plan", "operator": "equals", "value": "pro"},
					{"type": "optIn", "status": "accepted"},
					{"type": "activity", "action": "opened", "negate": false, "target": "campaign", "id": "cmp_1"}
				]
			}
		}`
		serveJSON(t, http.StatusOK, body)

		s, err := runAudienceSegmentsGet(cfg(t), "seg_abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ID != "seg_abc" {
			t.Errorf("ID = %q, want seg_abc", s.ID)
		}
		if deref(s.Description) != "Pro plan + opted in + active" {
			t.Errorf("Description = %q", deref(s.Description))
		}
		if s.Filter == nil {
			t.Fatal("Filter nil")
		}
		if s.Filter.Match != "all" {
			t.Errorf("Match = %q", s.Filter.Match)
		}
		if len(s.Filter.Conditions) != 3 {
			t.Fatalf("Conditions len = %d, want 3", len(s.Filter.Conditions))
		}
		if s.Filter.Conditions[0].Type != loops.AudienceConditionTypeProperty || s.Filter.Conditions[0].Property == nil {
			t.Errorf("property condition not decoded: %+v", s.Filter.Conditions[0])
		}
		if s.Filter.Conditions[1].Type != loops.AudienceConditionTypeOptIn || s.Filter.Conditions[1].OptIn == nil {
			t.Errorf("optIn condition not decoded: %+v", s.Filter.Conditions[1])
		}
		if s.Filter.Conditions[2].Type != loops.AudienceConditionTypeActivity || s.Filter.Conditions[2].Activity == nil {
			t.Errorf("activity condition not decoded: %+v", s.Filter.Conditions[2])
		}
	})

	t.Run("nil filter (all-contacts reserved segment)", func(t *testing.T) {
		body := `{
			"id": "seg_all",
			"name": "All contacts",
			"description": null,
			"createdAt": "2026-04-01T10:00:00Z",
			"updatedAt": "2026-04-01T10:00:00Z",
			"filter": null
		}`
		serveJSON(t, http.StatusOK, body)

		s, err := runAudienceSegmentsGet(cfg(t), "seg_all")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Filter != nil {
			t.Errorf("Filter = %+v, want nil", s.Filter)
		}
		if got := formatSegmentFilter(s.Filter); got != "(all contacts)" {
			t.Errorf("formatSegmentFilter = %q, want (all contacts)", got)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Audience segment not found"}`)
		_, err := runAudienceSegmentsGet(cfg(t), "seg_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
