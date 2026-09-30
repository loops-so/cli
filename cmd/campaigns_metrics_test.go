package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const campaignMetricsBody = `{"sends":4210,"opens":1922,"clicks":301,"unsubscribes":12,"spamReports":1,"hardBounces":9,"softBounces":13}`

func TestRunCampaignsMetrics(t *testing.T) {
	t.Run("returns the metrics", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, campaignMetricsBody)
		m, err := runCampaignsMetrics(cfg(t), "cmp_abc123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cap.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", cap.Method)
		}
		if cap.Path != "/campaigns/cmp_abc123/metrics" {
			t.Errorf("Path = %q, want /campaigns/cmp_abc123/metrics", cap.Path)
		}
		if m.Sends != 4210 {
			t.Errorf("Sends = %d, want 4210", m.Sends)
		}
		if m.Opens != 1922 {
			t.Errorf("Opens = %d, want 1922", m.Opens)
		}
		if m.Clicks != 301 {
			t.Errorf("Clicks = %d, want 301", m.Clicks)
		}
		if m.Unsubscribes != 12 {
			t.Errorf("Unsubscribes = %d, want 12", m.Unsubscribes)
		}
		if m.SpamReports != 1 {
			t.Errorf("SpamReports = %d, want 1", m.SpamReports)
		}
		if m.HardBounces != 9 {
			t.Errorf("HardBounces = %d, want 9", m.HardBounces)
		}
		if m.SoftBounces != 13 {
			t.Errorf("SoftBounces = %d, want 13", m.SoftBounces)
		}
	})

	t.Run("returns the api error for an unsent campaign", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"success":false,"message":"Campaign has not been sent."}`)
		_, err := runCampaignsMetrics(cfg(t), "cmp_draft")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Campaign has not been sent.") {
			t.Errorf("error = %q, want the API message", err.Error())
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Campaign not found."}`)
		_, err := runCampaignsMetrics(cfg(t), "cmp_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestCampaignsMetricsCmd(t *testing.T) {
	t.Run("prints the metrics table", func(t *testing.T) {
		useOutputFormat(t, "text")
		serveJSON(t, http.StatusOK, campaignMetricsBody)

		var out strings.Builder
		cmd := *campaignsMetricsCmd
		cmd.SetOut(&out)
		if err := cmd.RunE(&cmd, []string{"cmp_abc123"}); err != nil {
			t.Fatalf("RunE: %v", err)
		}
		got := fieldValueRows(out.String())
		if got["sends"] != "4210" || got["softBounces"] != "13" {
			t.Errorf("rows = %v\n%s", got, out.String())
		}
	})

	t.Run("prints json", func(t *testing.T) {
		useOutputFormat(t, "json")
		serveJSON(t, http.StatusOK, campaignMetricsBody)

		var out strings.Builder
		cmd := *campaignsMetricsCmd
		cmd.SetOut(&out)
		if err := cmd.RunE(&cmd, []string{"cmp_abc123"}); err != nil {
			t.Fatalf("RunE: %v", err)
		}
		var got map[string]int
		if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
			t.Fatalf("decode output: %v\n%s", err, out.String())
		}
		if len(got) != 7 || got["sends"] != 4210 || got["spamReports"] != 1 {
			t.Errorf("json = %v", got)
		}
	})
}
