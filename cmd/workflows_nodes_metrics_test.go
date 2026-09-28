package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const workflowNodeMetricsBody = `{"sends":4210,"opens":1922,"clicks":301,"unsubscribes":12,"spamReports":1,"hardBounces":9,"softBounces":13}`

func TestRunWorkflowsNodeMetrics(t *testing.T) {
	t.Run("returns the metrics", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, workflowNodeMetricsBody)
		m, err := runWorkflowsNodeMetrics(cfg(t), "wf_1", "node_s")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cap.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", cap.Method)
		}
		if cap.Path != "/workflows/wf_1/nodes/node_s/metrics" {
			t.Errorf("Path = %q, want /workflows/wf_1/nodes/node_s/metrics", cap.Path)
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

	t.Run("returns zero counters for a node that has not sent", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"sends":0,"opens":0,"clicks":0,"unsubscribes":0,"spamReports":0,"hardBounces":0,"softBounces":0}`)
		m, err := runWorkflowsNodeMetrics(cfg(t), "wf_1", "node_new")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Sends != 0 || m.Opens != 0 || m.SoftBounces != 0 {
			t.Errorf("metrics = %+v, want all zero", *m)
		}
	})

	t.Run("returns the api error for a non-email node", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"success":false,"message":"Node is not a SendEmailAction node."}`)
		_, err := runWorkflowsNodeMetrics(cfg(t), "wf_1", "node_timer")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Node is not a SendEmailAction node.") {
			t.Errorf("error = %q, want the API message", err.Error())
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Workflow not found."}`)
		_, err := runWorkflowsNodeMetrics(cfg(t), "wf_missing", "node_s")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestWorkflowsNodesMetricsCmd(t *testing.T) {
	t.Run("prints the metrics table", func(t *testing.T) {
		useOutputFormat(t, "text")
		serveJSON(t, http.StatusOK, workflowNodeMetricsBody)

		var out strings.Builder
		cmd := *workflowsNodesMetricsCmd
		cmd.SetOut(&out)
		if err := cmd.RunE(&cmd, []string{"wf_1", "node_s"}); err != nil {
			t.Fatalf("RunE: %v", err)
		}
		got := fieldValueRows(out.String())
		if got["sends"] != "4210" || got["clicks"] != "301" {
			t.Errorf("rows = %v\n%s", got, out.String())
		}
	})

	t.Run("prints json", func(t *testing.T) {
		useOutputFormat(t, "json")
		serveJSON(t, http.StatusOK, workflowNodeMetricsBody)

		var out strings.Builder
		cmd := *workflowsNodesMetricsCmd
		cmd.SetOut(&out)
		if err := cmd.RunE(&cmd, []string{"wf_1", "node_s"}); err != nil {
			t.Fatalf("RunE: %v", err)
		}
		var got map[string]int
		if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
			t.Fatalf("decode output: %v\n%s", err, out.String())
		}
		if len(got) != 7 || got["opens"] != 1922 {
			t.Errorf("json = %v", got)
		}
	})
}
