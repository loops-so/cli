package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const transactionalMetricsBody = `{"sends":4210,"deliveries":4175,"spamReports":1,"hardBounces":9,"softBounces":13}`

func TestRunTransactionalMetrics(t *testing.T) {
	t.Run("returns the metrics", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, transactionalMetricsBody)
		m, err := runTransactionalMetrics(cfg(t), "tx_abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cap.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", cap.Method)
		}
		if cap.Path != "/transactional-emails/tx_abc/metrics" {
			t.Errorf("Path = %q, want /transactional-emails/tx_abc/metrics", cap.Path)
		}
		if m.Sends != 4210 {
			t.Errorf("Sends = %d, want 4210", m.Sends)
		}
		if m.Deliveries != 4175 {
			t.Errorf("Deliveries = %d, want 4175", m.Deliveries)
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

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Transactional email not found."}`)
		_, err := runTransactionalMetrics(cfg(t), "tx_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestTransactionalMetricsCmd(t *testing.T) {
	t.Run("prints the metrics table", func(t *testing.T) {
		useOutputFormat(t, "text")
		serveJSON(t, http.StatusOK, transactionalMetricsBody)

		var out strings.Builder
		cmd := *transactionalMetricsCmd
		cmd.SetOut(&out)
		if err := cmd.RunE(&cmd, []string{"tx_abc"}); err != nil {
			t.Fatalf("RunE: %v", err)
		}
		got := fieldValueRows(out.String())
		if got["sends"] != "4210" || got["deliveries"] != "4175" {
			t.Errorf("rows = %v\n%s", got, out.String())
		}
	})

	t.Run("prints json", func(t *testing.T) {
		useOutputFormat(t, "json")
		serveJSON(t, http.StatusOK, transactionalMetricsBody)

		var out strings.Builder
		cmd := *transactionalMetricsCmd
		cmd.SetOut(&out)
		if err := cmd.RunE(&cmd, []string{"tx_abc"}); err != nil {
			t.Fatalf("RunE: %v", err)
		}
		var got map[string]int
		if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
			t.Fatalf("decode output: %v\n%s", err, out.String())
		}
		if len(got) != 5 || got["sends"] != 4210 || got["deliveries"] != 4175 {
			t.Errorf("json = %v", got)
		}
	})
}
