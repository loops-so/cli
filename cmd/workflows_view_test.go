package cmd

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

const workflowViewBody = `{
	"id": "wf_123",
	"url": "https://app.loops.so/workflows/wf_123",
	"status": "Draft",
	"workflowRevisionId": "rev_1",
	"name": "Welcome",
	"nodes": {}
}`

func TestWorkflowsViewCmd(t *testing.T) {
	t.Run("opens the workflow url", func(t *testing.T) {
		useOutputFormat(t, "text")
		opened := stubBrowser(t)
		cap := serveJSONCapture(t, http.StatusOK, workflowViewBody)

		var out strings.Builder
		cmd := *workflowsViewCmd
		cmd.SetOut(&out)
		if err := cmd.RunE(&cmd, []string{"wf_123"}); err != nil {
			t.Fatalf("RunE: %v", err)
		}
		if cap.Path != "/workflows/wf_123" {
			t.Errorf("Path = %q, want /workflows/wf_123", cap.Path)
		}
		if *opened != "https://app.loops.so/workflows/wf_123" {
			t.Errorf("opened = %q", *opened)
		}
		if !strings.Contains(out.String(), "https://app.loops.so/workflows/wf_123") {
			t.Errorf("output = %q, want the url", out.String())
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		opened := stubBrowser(t)
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Workflow not found"}`)

		cmd := *workflowsViewCmd
		cmd.SetOut(io.Discard)
		if err := cmd.RunE(&cmd, []string{"wf_missing"}); err == nil {
			t.Fatal("expected error, got nil")
		}
		if *opened != "" {
			t.Errorf("browser opened %q, want no open", *opened)
		}
	})
}

func TestWorkflowsGetCmdWeb(t *testing.T) {
	useOutputFormat(t, "text")
	opened := stubBrowser(t)
	serveJSON(t, http.StatusOK, workflowViewBody)

	var out strings.Builder
	cmd := *workflowsGetCmd
	cmd.SetOut(&out)
	if err := cmd.ParseFlags([]string{"--web"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	t.Cleanup(func() { cmd.Flags().Set("web", "false") })
	if err := cmd.RunE(&cmd, []string{"wf_123"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if *opened != "https://app.loops.so/workflows/wf_123" {
		t.Errorf("opened = %q", *opened)
	}
	if strings.Contains(out.String(), "workflowId") {
		t.Errorf("output printed the detail table with --web:\n%s", out.String())
	}
}
