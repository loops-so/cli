package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunComponentsUpdate(t *testing.T) {
	body := `{
		"id": "cmp_1",
		"name": "Footer",
		"lmx": "<Paragraph>Updated</Paragraph>",
		"affectedEmailCount": 3
	}`

	t.Run("returns result with affectedEmailCount on success", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		result, err := runComponentsUpdate(cfg(t), "cmp_1", loops.UpdateComponentRequest{LMX: "<Paragraph>Updated</Paragraph>"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != "cmp_1" {
			t.Errorf("ID = %q, want cmp_1", result.ID)
		}
		if result.AffectedEmailCount != 3 {
			t.Errorf("AffectedEmailCount = %d, want 3", result.AffectedEmailCount)
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"not found"}`)
		_, err := runComponentsUpdate(cfg(t), "cmp_missing", loops.UpdateComponentRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("sends only name when only name set", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, body)
		_, err := runComponentsUpdate(cfg(t), "cmp_1", loops.UpdateComponentRequest{Name: "Renamed"})
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
		if _, ok := sent["lmx"]; ok {
			t.Errorf("lmx should be omitted when unset, got %v", sent["lmx"])
		}
	})
}

func TestComponentsUpdateCmdLmxFileAndAffectedCount(t *testing.T) {
	body := `{"id":"cmp_1","name":"Footer","lmx":"<Paragraph>From file</Paragraph>","affectedEmailCount":7}`
	got := serveJSONCapture(t, http.StatusOK, body)

	dir := t.TempDir()
	path := filepath.Join(dir, "footer.lmx")
	if err := os.WriteFile(path, []byte("<Paragraph>From file</Paragraph>"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	saved := outputFormat
	outputFormat = "text"
	t.Cleanup(func() { outputFormat = saved })

	var out bytes.Buffer
	cmd := *componentsUpdateCmd
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.ParseFlags([]string{"--lmx-file", path}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := cmd.RunE(&cmd, []string{"cmp_1"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	var sent map[string]any
	if err := json.Unmarshal(got.Body, &sent); err != nil {
		t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
	}
	if sent["lmx"] != "<Paragraph>From file</Paragraph>" {
		t.Errorf("lmx = %v, want file contents", sent["lmx"])
	}

	if !strings.Contains(out.String(), "affectedEmailCount: 7") {
		t.Errorf("output missing affectedEmailCount: 7\ngot:\n%s", out.String())
	}
}
