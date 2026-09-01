package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunWorkflowsDelete(t *testing.T) {
	t.Run("sends expectedRevisionId and omits confirmDelete when false", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusNoContent, "")
		rev := "rev_1"
		if err := runWorkflowsDelete(cfg(t), "wf_abc", loops.DeleteWorkflowRequest{ExpectedRevisionID: &rev}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cap.Method != http.MethodDelete {
			t.Errorf("Method = %q, want DELETE", cap.Method)
		}
		if cap.Path != "/workflows/wf_abc" {
			t.Errorf("Path = %q, want /workflows/wf_abc", cap.Path)
		}

		var body map[string]any
		if err := json.Unmarshal(cap.Body, &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if body["expectedRevisionId"] != "rev_1" {
			t.Errorf("expectedRevisionId = %v, want rev_1", body["expectedRevisionId"])
		}
		if _, ok := body["confirmDelete"]; ok {
			t.Errorf("confirmDelete present, want omitted")
		}
	})

	t.Run("sends null revision and confirmDelete when confirmed", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusNoContent, "")
		if err := runWorkflowsDelete(cfg(t), "wf_abc", loops.DeleteWorkflowRequest{ConfirmDelete: true}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var body map[string]any
		if err := json.Unmarshal(cap.Body, &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		v, ok := body["expectedRevisionId"]
		if !ok || v != nil {
			t.Errorf("expectedRevisionId = %v (present %v), want JSON null", v, ok)
		}
		if body["confirmDelete"] != true {
			t.Errorf("confirmDelete = %v, want true", body["confirmDelete"])
		}
	})

	t.Run("reports confirmation required on 409 with the confirm sentence", func(t *testing.T) {
		serveJSON(t, http.StatusConflict, `{"success":false,"message":"This workflow is currently sending to 12 contacts. `+apiConfirmSentence+`"}`)
		err := runWorkflowsDelete(cfg(t), "wf_abc", loops.DeleteWorkflowRequest{})
		if !errors.Is(err, loops.ErrWorkflowDeleteConfirmationRequired) {
			t.Fatalf("error = %v, want ErrWorkflowDeleteConfirmationRequired", err)
		}
		var apiErr *loops.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("error does not unwrap to *loops.APIError: %v", err)
		}
		if apiErr.StatusCode != http.StatusConflict {
			t.Errorf("StatusCode = %d, want 409", apiErr.StatusCode)
		}
	})

	t.Run("a stale revision 409 is a plain API error", func(t *testing.T) {
		serveJSON(t, http.StatusConflict, `{"success":false,"message":"Workflow revision is out of date."}`)
		err := runWorkflowsDelete(cfg(t), "wf_abc", loops.DeleteWorkflowRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errors.Is(err, loops.ErrWorkflowDeleteConfirmationRequired) {
			t.Errorf("stale revision reported as confirmation required: %v", err)
		}
	})
}
