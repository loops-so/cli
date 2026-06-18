package cmd

import (
	"net/http"
	"testing"
)

func TestRunThemesGet(t *testing.T) {
	body := `{
		"success": true,
		"id": "thm_abc123",
		"name": "Default",
		"isDefault": true,
		"createdAt": "2026-04-01T10:00:00Z",
		"updatedAt": "2026-04-02T10:00:00Z",
		"styles": {
			"backgroundColor": "#ffffff",
			"bodyColor": "#000000",
			"textBaseFontSize": 16,
			"heading1FontSize": 32
		}
	}`

	t.Run("returns the theme", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		th, err := runThemesGet(cfg(t), "thm_abc123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if th.ID != "thm_abc123" {
			t.Errorf("ID = %q, want thm_abc123", th.ID)
		}
		if th.Name != "Default" {
			t.Errorf("Name = %q, want Default", th.Name)
		}
		if !th.IsDefault {
			t.Error("IsDefault = false, want true")
		}
		if th.Styles.BackgroundColor != "#ffffff" {
			t.Errorf("Styles.BackgroundColor = %q, want #ffffff", th.Styles.BackgroundColor)
		}
		if th.Styles.TextBaseFontSize != 16 {
			t.Errorf("Styles.TextBaseFontSize = %v, want 16", th.Styles.TextBaseFontSize)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Theme not found"}`)
		_, err := runThemesGet(cfg(t), "thm_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
