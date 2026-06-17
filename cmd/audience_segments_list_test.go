package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunAudienceSegmentsList(t *testing.T) {
	t.Run("returns segments", func(t *testing.T) {
		body := `{
			"pagination": {"nextCursor": ""},
			"data": [
				{
					"id": "seg_1",
					"name": "All contacts",
					"description": null,
					"createdAt": "2026-04-01T10:00:00Z",
					"updatedAt": "2026-04-01T10:00:00Z",
					"filter": null
				},
				{
					"id": "seg_2",
					"name": "Pro plan",
					"description": "Paying customers",
					"createdAt": "2026-04-02T10:00:00Z",
					"updatedAt": "2026-04-20T10:00:00Z",
					"filter": {
						"match": "all",
						"conditions": [
							{"type": "property", "key": "plan", "operator": "equals", "value": "pro"}
						]
					}
				}
			]
		}`
		serveJSON(t, http.StatusOK, body)

		segments, err := runAudienceSegmentsList(cfg(t), loops.PaginationParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(segments) != 2 {
			t.Fatalf("len = %d, want 2", len(segments))
		}
		if segments[0].ID != "seg_1" || segments[0].Filter != nil {
			t.Errorf("segments[0] = %+v", segments[0])
		}
		if segments[1].Filter == nil || len(segments[1].Filter.Conditions) != 1 {
			t.Errorf("segments[1].Filter = %+v", segments[1].Filter)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"error":"unauthorized"}`)
		_, err := runAudienceSegmentsList(cfg(t), loops.PaginationParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
