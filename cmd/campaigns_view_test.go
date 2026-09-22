package cmd

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

const campaignViewBody = `{
	"success": true,
	"id": "cmp_abc123",
	"url": "https://app.loops.so/campaigns/cmp_abc123",
	"emailMessageId": "em_abc123",
	"name": "Spring Launch",
	"status": "Draft",
	"createdAt": "2026-04-01T10:00:00Z",
	"updatedAt": "2026-04-02T10:00:00Z"
}`

func TestCampaignsViewCmd(t *testing.T) {
	t.Run("opens the campaign url", func(t *testing.T) {
		useOutputFormat(t, "text")
		opened := stubBrowser(t)
		cap := serveJSONCapture(t, http.StatusOK, campaignViewBody)

		var out strings.Builder
		cmd := *campaignsViewCmd
		cmd.SetOut(&out)
		if err := cmd.RunE(&cmd, []string{"cmp_abc123"}); err != nil {
			t.Fatalf("RunE: %v", err)
		}
		if cap.Path != "/campaigns/cmp_abc123" {
			t.Errorf("Path = %q, want /campaigns/cmp_abc123", cap.Path)
		}
		if *opened != "https://app.loops.so/campaigns/cmp_abc123" {
			t.Errorf("opened = %q", *opened)
		}
		if !strings.Contains(out.String(), "https://app.loops.so/campaigns/cmp_abc123") {
			t.Errorf("output = %q, want the url", out.String())
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		opened := stubBrowser(t)
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Campaign not found"}`)

		cmd := *campaignsViewCmd
		cmd.SetOut(io.Discard)
		if err := cmd.RunE(&cmd, []string{"cmp_missing"}); err == nil {
			t.Fatal("expected error, got nil")
		}
		if *opened != "" {
			t.Errorf("browser opened %q, want no open", *opened)
		}
	})
}

func TestCampaignsGetCmdWeb(t *testing.T) {
	useOutputFormat(t, "text")
	opened := stubBrowser(t)
	serveJSON(t, http.StatusOK, campaignViewBody)

	var out strings.Builder
	cmd := *campaignsGetCmd
	cmd.SetOut(&out)
	if err := cmd.ParseFlags([]string{"--web"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	t.Cleanup(func() { cmd.Flags().Set("web", "false") })
	if err := cmd.RunE(&cmd, []string{"cmp_abc123"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if *opened != "https://app.loops.so/campaigns/cmp_abc123" {
		t.Errorf("opened = %q", *opened)
	}
	if strings.Contains(out.String(), "campaignId") {
		t.Errorf("output printed the detail table with --web:\n%s", out.String())
	}
}
