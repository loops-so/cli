package cmd

import (
	"net/http"
	"testing"

	loops "github.com/loops-so/loops-go"
)

func TestRunEmailMessageGuardian(t *testing.T) {
	body := `{
		"errors": [
			{
				"rule": "missing-unsubscribe",
				"title": "Missing unsubscribe link",
				"description": "Marketing emails must include an unsubscribe link.",
				"items": [
					{"label": "Footer", "codeName": "footer"}
				]
			}
		],
		"warnings": [
			{
				"rule": "broken-link",
				"title": "Broken link",
				"description": "One or more links may be broken.",
				"items": [
					{"label": "https://example.com/a", "codeName": "link-a"},
					{"label": "https://example.com/b", "codeName": "link-b"}
				]
			}
		]
	}`

	t.Run("returns errors and warnings", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		resp, err := runEmailMessageGuardian(cfg(t), "em_abc123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Errors) != 1 {
			t.Fatalf("Errors len = %d, want 1", len(resp.Errors))
		}
		if resp.Errors[0].Rule != "missing-unsubscribe" {
			t.Errorf("Errors[0].Rule = %q, want missing-unsubscribe", resp.Errors[0].Rule)
		}
		if len(resp.Errors[0].Items) != 1 || resp.Errors[0].Items[0].Label != "Footer" {
			t.Errorf("Errors[0].Items = %+v, want one item labeled Footer", resp.Errors[0].Items)
		}
		if len(resp.Warnings) != 1 {
			t.Fatalf("Warnings len = %d, want 1", len(resp.Warnings))
		}
		if len(resp.Warnings[0].Items) != 2 {
			t.Errorf("Warnings[0].Items len = %d, want 2", len(resp.Warnings[0].Items))
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Email message not found"}`)
		_, err := runEmailMessageGuardian(cfg(t), "em_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFormatGuardianItems(t *testing.T) {
	tests := []struct {
		name  string
		items []loops.GuardianRuleItem
		want  string
	}{
		{"no items", nil, "0"},
		{"label", []loops.GuardianRuleItem{{Label: "Footer"}}, "1 (Footer)"},
		{"codeName fallback", []loops.GuardianRuleItem{{CodeName: "footer"}}, "1 (footer)"},
		{"empty label and codeName", []loops.GuardianRuleItem{{}}, "1"},
		{"multiple", []loops.GuardianRuleItem{{Label: "A"}, {Label: "B"}}, "2 (A, B)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatGuardianItems(tc.items)
			if got != tc.want {
				t.Errorf("formatGuardianItems() = %q, want %q", got, tc.want)
			}
		})
	}
}
