package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func TestRunComponentsCreate(t *testing.T) {
	body := `{
		"id": "cmp_new",
		"name": "Footer",
		"lmx": "<Paragraph>Hi</Paragraph>"
	}`

	t.Run("returns component on success", func(t *testing.T) {
		serveJSON(t, http.StatusCreated, body)
		c, err := runComponentsCreate(cfg(t), loops.CreateComponentRequest{Name: "Footer", LMX: "<Paragraph>Hi</Paragraph>"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.ID != "cmp_new" {
			t.Errorf("ID = %q, want cmp_new", c.ID)
		}
		if c.Name != "Footer" {
			t.Errorf("Name = %q, want Footer", c.Name)
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"success":false,"message":"name is required"}`)
		_, err := runComponentsCreate(cfg(t), loops.CreateComponentRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("sends name and lmx in request body", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, body)
		_, err := runComponentsCreate(cfg(t), loops.CreateComponentRequest{
			Name: "Footer",
			LMX:  "<Paragraph>Hi</Paragraph>",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sent map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
		}
		if sent["name"] != "Footer" {
			t.Errorf("name = %v, want Footer", sent["name"])
		}
		if sent["lmx"] != "<Paragraph>Hi</Paragraph>" {
			t.Errorf("lmx = %v", sent["lmx"])
		}
	})
}

func TestComponentsCreateCmdLmxFile(t *testing.T) {
	body := `{"id":"cmp_new","name":"Footer","lmx":"<Paragraph>From file</Paragraph>"}`
	got := serveJSONCapture(t, http.StatusCreated, body)

	dir := t.TempDir()
	path := filepath.Join(dir, "footer.lmx")
	if err := os.WriteFile(path, []byte("<Paragraph>From file</Paragraph>"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	saved := outputFormat
	outputFormat = "text"
	t.Cleanup(func() { outputFormat = saved })

	cmd := *componentsCreateCmd
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.ParseFlags([]string{"--name", "Footer", "--lmx-file", path}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := cmd.RunE(&cmd, []string{}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	var sent map[string]any
	if err := json.Unmarshal(got.Body, &sent); err != nil {
		t.Fatalf("decode request body: %v\nraw: %s", err, got.Body)
	}
	if sent["lmx"] != "<Paragraph>From file</Paragraph>" {
		t.Errorf("lmx = %v, want file contents", sent["lmx"])
	}
	if sent["name"] != "Footer" {
		t.Errorf("name = %v, want Footer", sent["name"])
	}
}

// newLmxTestCmd returns a fresh command carrying its own flag set so subtests
// do not alias the shared package-level command's flags.
func newLmxTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("lmx", "", "")
	cmd.Flags().String("lmx-file", "", "")
	return cmd
}

func TestLmxFromCmd(t *testing.T) {
	t.Run("inline lmx", func(t *testing.T) {
		cmd := newLmxTestCmd()
		cmd.ParseFlags([]string{"--lmx", "<Paragraph>Inline</Paragraph>"})
		lmx, set, err := lmxFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !set || lmx != "<Paragraph>Inline</Paragraph>" {
			t.Errorf("lmx=%q set=%v", lmx, set)
		}
	})

	t.Run("missing lmx-file returns error", func(t *testing.T) {
		cmd := newLmxTestCmd()
		cmd.ParseFlags([]string{"--lmx-file", "/does/not/exist.lmx"})
		if _, _, err := lmxFromCmd(cmd); err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("neither flag set", func(t *testing.T) {
		cmd := newLmxTestCmd()
		cmd.ParseFlags([]string{"--name", "x"})
		lmx, set, err := lmxFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if set || lmx != "" {
			t.Errorf("expected unset, got lmx=%q set=%v", lmx, set)
		}
	})
}
