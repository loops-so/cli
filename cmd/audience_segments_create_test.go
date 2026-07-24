package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/loops-so/loops-go"
)

const audienceSegmentBody = `{
	"id": "seg_new",
	"name": "Active users",
	"description": "Recently active",
	"createdAt": "2026-07-24T10:00:00Z",
	"updatedAt": "2026-07-24T10:00:00Z",
	"filter": {"match": "all", "conditions": []}
}`

func strPtr(s string) *string { return &s }

func writeFilterFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "filter.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write filter file: %v", err)
	}
	return path
}

func TestRunAudienceSegmentsCreate(t *testing.T) {
	t.Run("returns segment on success", func(t *testing.T) {
		serveJSON(t, http.StatusOK, audienceSegmentBody)
		s, err := runAudienceSegmentsCreate(cfg(t), loops.CreateAudienceSegmentRequest{
			Name:   "Active users",
			Filter: loops.AudienceFilter{Match: "all"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ID != "seg_new" {
			t.Errorf("ID = %q, want seg_new", s.ID)
		}
		if s.Name != "Active users" {
			t.Errorf("Name = %q, want Active users", s.Name)
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"success":false,"message":"name is required"}`)
		_, err := runAudienceSegmentsCreate(cfg(t), loops.CreateAudienceSegmentRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("sends name, description and filter", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, audienceSegmentBody)
		_, err := runAudienceSegmentsCreate(cfg(t), loops.CreateAudienceSegmentRequest{
			Name:        "Active users",
			Description: "Recently active",
			Filter: loops.AudienceFilter{
				Match: "all",
				Conditions: []loops.AudienceFilterCondition{
					{
						Type: "property",
						Property: &loops.PropertyCondition{
							Key:      "plan",
							Operator: "eq",
							Value:    &loops.PropertyConditionValue{String: strPtr("pro")},
						},
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
		if sent["name"] != "Active users" {
			t.Errorf("name = %v, want Active users", sent["name"])
		}
		if sent["description"] != "Recently active" {
			t.Errorf("description = %v, want Recently active", sent["description"])
		}
		filter, ok := sent["filter"].(map[string]any)
		if !ok {
			t.Fatalf("filter not an object: %v", sent["filter"])
		}
		if filter["match"] != "all" {
			t.Errorf("filter.match = %v, want all", filter["match"])
		}
		conds, ok := filter["conditions"].([]any)
		if !ok || len(conds) != 1 {
			t.Fatalf("filter.conditions = %v, want 1 condition", filter["conditions"])
		}
		cond, ok := conds[0].(map[string]any)
		if !ok {
			t.Fatalf("condition not an object: %v", conds[0])
		}
		if cond["type"] != "property" || cond["key"] != "plan" || cond["value"] != "pro" {
			t.Errorf("condition = %v, want type=property key=plan value=pro", cond)
		}
	})
}

func TestAudienceSegmentsCreateCmd(t *testing.T) {
	t.Run("errors when filter file is missing", func(t *testing.T) {
		serveJSON(t, http.StatusOK, audienceSegmentBody)
		cmd := audienceSegmentsCreateCmd
		cmd.SetArgs([]string{})
		cmd.Flags().Set("name", "Active users")
		cmd.Flags().Set("filter-file", filepath.Join(t.TempDir(), "does-not-exist.json"))
		t.Cleanup(func() { cmd.Flags().Set("filter-file", "") })
		if err := cmd.RunE(cmd, nil); err == nil {
			t.Fatal("expected error for missing filter file, got nil")
		}
	})

	t.Run("errors on invalid filter JSON", func(t *testing.T) {
		serveJSON(t, http.StatusOK, audienceSegmentBody)
		path := writeFilterFile(t, `{not json`)
		cmd := audienceSegmentsCreateCmd
		cmd.SetArgs([]string{})
		cmd.Flags().Set("name", "Active users")
		cmd.Flags().Set("filter-file", path)
		t.Cleanup(func() { cmd.Flags().Set("filter-file", "") })
		if err := cmd.RunE(cmd, nil); err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})

	t.Run("succeeds with a valid filter file", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, audienceSegmentBody)
		path := writeFilterFile(t, `{"match":"any","conditions":[]}`)
		cmd := audienceSegmentsCreateCmd
		cmd.SetArgs([]string{})
		cmd.Flags().Set("name", "Active users")
		cmd.Flags().Set("filter-file", path)
		t.Cleanup(func() {
			cmd.Flags().Set("filter-file", "")
			cmd.Flags().Set("name", "")
			cmd.Flags().Set("description", "")
		})
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		filter, ok := sent["filter"].(map[string]any)
		if !ok || filter["match"] != "any" {
			t.Errorf("filter = %v, want match=any", sent["filter"])
		}
	})

	t.Run("succeeds with an inline --filter", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, audienceSegmentBody)
		cmd := audienceSegmentsCreateCmd
		cmd.SetArgs([]string{})
		cmd.Flags().Set("name", "Active users")
		cmd.Flags().Set("filter", `{"match":"all","conditions":[]}`)
		t.Cleanup(func() {
			cmd.Flags().Set("filter", "")
			cmd.Flags().Set("name", "")
			cmd.Flags().Set("description", "")
		})
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		filter, ok := sent["filter"].(map[string]any)
		if !ok || filter["match"] != "all" {
			t.Errorf("filter = %v, want match=all", sent["filter"])
		}
	})
}
