package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunWorkflowsGet(t *testing.T) {
	body := `{
		"id": "wf_123",
		"name": "Welcome",
		"description": "New user welcome series",
		"emoji": "👋",
		"mailingListId": null,
		"rootNodeId": "node_root",
		"nodes": {
			"node_root": {
				"typeName": "SignupTrigger",
				"nextNodeIds": ["node_email"]
			},
			"node_email": {
				"typeName": "SendEmailAction",
				"nextNodeIds": [],
				"emailMessageId": "em_1",
				"subject": "Hello"
			}
		}
	}`

	t.Run("returns the workflow", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		w, err := runWorkflowsGet(cfg(t), "wf_123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.ID != "wf_123" {
			t.Errorf("ID = %q, want wf_123", w.ID)
		}
		if w.Name != "Welcome" {
			t.Errorf("Name = %q, want Welcome", w.Name)
		}
		if w.MailingListID != nil {
			t.Errorf("MailingListID = %v, want nil", w.MailingListID)
		}
		if deref(w.RootNodeID) != "node_root" {
			t.Errorf("RootNodeID = %q, want node_root", deref(w.RootNodeID))
		}
		if len(w.Nodes) != 2 {
			t.Fatalf("expected 2 nodes, got %d", len(w.Nodes))
		}
		root := w.Nodes["node_root"]
		if root.TypeName != loops.WorkflowNodeTypeSignupTrigger {
			t.Errorf("root.TypeName = %q, want SignupTrigger", root.TypeName)
		}
		if root.SignupTrigger == nil {
			t.Fatal("root.SignupTrigger = nil, want populated")
		}
		email := w.Nodes["node_email"]
		if email.SendEmailAction == nil {
			t.Fatal("email.SendEmailAction = nil, want populated")
		}
		if email.SendEmailAction.EmailMessageID != "em_1" {
			t.Errorf("emailMessageId = %q, want em_1", email.SendEmailAction.EmailMessageID)
		}
		if cap.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", cap.Method)
		}
		if cap.Path != "/workflows/wf_123" {
			t.Errorf("Path = %q, want /workflows/wf_123", cap.Path)
		}
	})

	t.Run("simplifiedNodeNextIDs extracts next IDs", func(t *testing.T) {
		serveJSON(t, http.StatusOK, body)
		w, err := runWorkflowsGet(cfg(t), "wf_123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := simplifiedNodeNextIDs(w.Nodes["node_root"])
		if len(got) != 1 || got[0] != "node_email" {
			t.Errorf("simplifiedNodeNextIDs = %v, want [node_email]", got)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Workflow not found"}`)
		_, err := runWorkflowsGet(cfg(t), "wf_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
