package cmd

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

const transactionalViewBody = `{
	"id": "tx_abc",
	"url": "https://app.loops.so/transactional/tx_abc",
	"name": "Welcome",
	"draftEmailMessageId": null,
	"publishedEmailMessageId": null,
	"createdAt": "2026-01-01T00:00:00Z",
	"updatedAt": "2026-01-02T00:00:00Z",
	"dataVariables": []
}`

func TestTransactionalViewCmd(t *testing.T) {
	t.Run("opens the transactional url", func(t *testing.T) {
		useOutputFormat(t, "text")
		opened := stubBrowser(t)
		cap := serveJSONCapture(t, http.StatusOK, transactionalViewBody)

		var out strings.Builder
		cmd := *transactionalViewCmd
		cmd.SetOut(&out)
		if err := cmd.RunE(&cmd, []string{"tx_abc"}); err != nil {
			t.Fatalf("RunE: %v", err)
		}
		if cap.Path != "/transactional-emails/tx_abc" {
			t.Errorf("Path = %q, want /transactional-emails/tx_abc", cap.Path)
		}
		if *opened != "https://app.loops.so/transactional/tx_abc" {
			t.Errorf("opened = %q", *opened)
		}
		if !strings.Contains(out.String(), "https://app.loops.so/transactional/tx_abc") {
			t.Errorf("output = %q, want the url", out.String())
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		opened := stubBrowser(t)
		serveJSON(t, http.StatusNotFound, `{"error":"not found"}`)

		cmd := *transactionalViewCmd
		cmd.SetOut(io.Discard)
		if err := cmd.RunE(&cmd, []string{"tx_missing"}); err == nil {
			t.Fatal("expected error, got nil")
		}
		if *opened != "" {
			t.Errorf("browser opened %q, want no open", *opened)
		}
	})
}

func TestTransactionalGetCmdWeb(t *testing.T) {
	useOutputFormat(t, "text")
	opened := stubBrowser(t)
	serveJSON(t, http.StatusOK, transactionalViewBody)

	var out strings.Builder
	cmd := *transactionalGetCmd
	cmd.SetOut(&out)
	if err := cmd.ParseFlags([]string{"--web"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	t.Cleanup(func() { cmd.Flags().Set("web", "false") })
	if err := cmd.RunE(&cmd, []string{"tx_abc"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if *opened != "https://app.loops.so/transactional/tx_abc" {
		t.Errorf("opened = %q", *opened)
	}
	if strings.Contains(out.String(), "transactionalId") {
		t.Errorf("output printed the detail table with --web:\n%s", out.String())
	}
}
