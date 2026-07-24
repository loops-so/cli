package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunThemesUpdate(t *testing.T) {
	body := `{
		"id": "theme_abc123",
		"name": "Renamed",
		"isDefault": false,
		"createdAt": "2026-04-01T10:00:00Z",
		"updatedAt": "2026-04-25T10:00:00Z",
		"styles": {"backgroundColor": "#000000"}
	}`

	t.Run("returns theme on success", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		th, err := runThemesUpdate(cfg(t), "theme_abc123", loops.UpdateThemeRequest{Name: "Renamed"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if th.ID != "theme_abc123" {
			t.Errorf("ID = %q, want theme_abc123", th.ID)
		}
		if th.Name != "Renamed" {
			t.Errorf("Name = %q, want Renamed", th.Name)
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"theme not found"}`)
		_, err := runThemesUpdate(cfg(t), "theme_missing", loops.UpdateThemeRequest{Name: "Renamed"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("sends name only when styles unset", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, body)
		_, err := runThemesUpdate(cfg(t), "theme_abc123", loops.UpdateThemeRequest{Name: "Renamed"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		if sent["name"] != "Renamed" {
			t.Errorf("name = %v, want Renamed", sent["name"])
		}
		if _, present := sent["styles"]; present {
			t.Errorf("styles should be omitted, got %v", sent["styles"])
		}
	})

	t.Run("sends styles under styles key", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, body)
		_, err := runThemesUpdate(cfg(t), "theme_abc123", loops.UpdateThemeRequest{
			Styles: &loops.ThemeStyles{BackgroundColor: "#000000"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		styles, ok := sent["styles"].(map[string]any)
		if !ok {
			t.Fatalf("styles not an object: %v", sent["styles"])
		}
		if styles["backgroundColor"] != "#000000" {
			t.Errorf("styles.backgroundColor = %v, want #000000", styles["backgroundColor"])
		}
	})
}
