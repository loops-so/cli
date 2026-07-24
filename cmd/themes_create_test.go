package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunThemesCreate(t *testing.T) {
	body := `{
		"id": "theme_new",
		"name": "Brand",
		"isDefault": false,
		"createdAt": "2026-04-20T10:00:00Z",
		"updatedAt": "2026-04-20T10:00:00Z",
		"styles": {"backgroundColor": "#ffffff", "borderWidth": 2}
	}`

	t.Run("returns theme on success", func(t *testing.T) {
		serveJSON(t, http.StatusCreated, body)
		th, err := runThemesCreate(cfg(t), loops.CreateThemeRequest{Name: "Brand"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if th.ID != "theme_new" {
			t.Errorf("ID = %q, want theme_new", th.ID)
		}
		if th.Name != "Brand" {
			t.Errorf("Name = %q, want Brand", th.Name)
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"success":false,"message":"name is required"}`)
		_, err := runThemesCreate(cfg(t), loops.CreateThemeRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("sends name and styles", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, body)
		_, err := runThemesCreate(cfg(t), loops.CreateThemeRequest{
			Name: "Brand",
			Styles: &loops.ThemeStyles{
				BackgroundColor: "#ffffff",
				BorderWidth:     2,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		if sent["name"] != "Brand" {
			t.Errorf("name = %v, want Brand", sent["name"])
		}
		styles, ok := sent["styles"].(map[string]any)
		if !ok {
			t.Fatalf("styles not an object: %v", sent["styles"])
		}
		if styles["backgroundColor"] != "#ffffff" {
			t.Errorf("styles.backgroundColor = %v, want #ffffff", styles["backgroundColor"])
		}
		if styles["borderWidth"] != float64(2) {
			t.Errorf("styles.borderWidth = %v, want 2", styles["borderWidth"])
		}
	})

	t.Run("omits styles when nil", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, body)
		_, err := runThemesCreate(cfg(t), loops.CreateThemeRequest{Name: "Brand"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		if _, present := sent["styles"]; present {
			t.Errorf("styles should be omitted, got %v", sent["styles"])
		}
	})
}
