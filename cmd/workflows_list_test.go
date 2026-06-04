package cmd

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunWorkflowsList(t *testing.T) {
	t.Run("returns workflows", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"pagination":{"nextCursor":""},"data":[{"id":"wf_1","name":"Onboarding","createdAt":"2026-04-01","updatedAt":"2026-04-02"}]}`)
		workflows, err := runWorkflowsList(cfg(t), loops.PaginationParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []WorkflowListItem{
			{ID: "wf_1", Name: "Onboarding", CreatedAt: "2026-04-01", UpdatedAt: "2026-04-02"},
		}
		if !reflect.DeepEqual(workflows, want) {
			t.Errorf("got %+v, want %+v", workflows, want)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"message":"unauthorized"}`)
		_, err := runWorkflowsList(cfg(t), loops.PaginationParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunWorkflowsGet(t *testing.T) {
	t.Run("returns workflow detail", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"id":"wf_1","name":"Onboarding","workflowRevisionId":"wf_rev_1","rootId":"node_0","emailMessages":[{"id":"em_1","emailMessageId":"em_1","nodeId":"node_1","nodeIds":["node_1"],"subject":"Welcome","previewText":"Start here","fromName":"Loops","fromEmail":"hello","replyToEmail":"","contentRevisionId":"rev_1","createdAt":"2026-04-01","updatedAt":"2026-04-02"}],"nodes":{"node_1":{"typeName":"SendEmailAction","nextNodeIds":[],"emailMessageId":"em_1"}}}`)
		workflow, err := runWorkflowsGet(cfg(t), "wf_1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if workflow.ID != "wf_1" {
			t.Fatalf("got workflow ID %q, want wf_1", workflow.ID)
		}
		if len(workflow.EmailMessages) != 1 {
			t.Fatalf("got %d email messages, want 1", len(workflow.EmailMessages))
		}
		message := workflow.EmailMessages[0]
		if message.EmailMessageID != "em_1" {
			t.Errorf("got emailMessageId %q, want em_1", message.EmailMessageID)
		}
		if deref(message.NodeID) != "node_1" {
			t.Errorf("got nodeId %q, want node_1", deref(message.NodeID))
		}
		if deref(message.ContentRevisionID) != "rev_1" {
			t.Errorf("got contentRevisionId %q, want rev_1", deref(message.ContentRevisionID))
		}
		node := workflow.Nodes["node_1"]
		if node.EmailMessageID != "em_1" {
			t.Errorf("got node emailMessageId %q, want em_1", node.EmailMessageID)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"message":"Workflow not found"}`)
		_, err := runWorkflowsGet(cfg(t), "wf_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
