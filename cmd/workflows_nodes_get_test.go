package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunWorkflowsNodeGet(t *testing.T) {
	t.Run("returns event trigger node", func(t *testing.T) {
		body := `{
			"typeName": "EventTrigger",
			"id": "node_abc",
			"workflowId": "wf_123",
			"nextNodeIds": ["node_next"],
			"eventName": "signup",
			"reEligible": true,
			"eventProperties": [
				{"name":"plan","type":"string"},
				{"name":"seats","type":"number"}
			]
		}`
		cap := serveJSONCapture(t, http.StatusOK, body)

		n, err := runWorkflowsNodeGet(cfg(t), "wf_123", "node_abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n.TypeName != loops.WorkflowNodeTypeEventTrigger {
			t.Errorf("TypeName = %q, want EventTrigger", n.TypeName)
		}
		if n.EventTrigger == nil {
			t.Fatal("EventTrigger = nil, want populated")
		}
		if deref(n.EventTrigger.EventName) != "signup" {
			t.Errorf("EventName = %q, want signup", deref(n.EventTrigger.EventName))
		}
		if !n.EventTrigger.ReEligible {
			t.Error("ReEligible = false, want true")
		}
		if len(n.EventTrigger.EventProperties) != 2 {
			t.Errorf("EventProperties = %d, want 2", len(n.EventTrigger.EventProperties))
		}
		if cap.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", cap.Method)
		}
		if cap.Path != "/workflows/wf_123/nodes/node_abc" {
			t.Errorf("Path = %q, want /workflows/wf_123/nodes/node_abc", cap.Path)
		}
	})

	t.Run("workflowNodeRows renders event trigger fields", func(t *testing.T) {
		eventName := "signup"
		n := &loops.WorkflowNode{
			TypeName: loops.WorkflowNodeTypeEventTrigger,
			EventTrigger: &loops.EventTriggerWorkflowNode{
				ID:          "node_abc",
				WorkflowID:  "wf_123",
				NextNodeIDs: []string{"a", "b"},
				EventName:   &eventName,
				ReEligible:  true,
				EventProperties: []loops.WorkflowEventProperty{
					{Name: "plan", Type: "string"},
				},
			},
		}
		rows := workflowNodeRows(n)
		got := map[string]string{}
		for _, r := range rows {
			got[r[0]] = r[1]
		}
		if got["typeName"] != "EventTrigger" {
			t.Errorf("typeName = %q, want EventTrigger", got["typeName"])
		}
		if got["nodeId"] != "node_abc" {
			t.Errorf("nodeId = %q, want node_abc", got["nodeId"])
		}
		if got["workflowId"] != "wf_123" {
			t.Errorf("workflowId = %q, want wf_123", got["workflowId"])
		}
		if got["nextNodeIds"] != "a, b" {
			t.Errorf("nextNodeIds = %q, want \"a, b\"", got["nextNodeIds"])
		}
		if got["eventName"] != "signup" {
			t.Errorf("eventName = %q, want signup", got["eventName"])
		}
		if got["reEligible"] != "true" {
			t.Errorf("reEligible = %q, want true", got["reEligible"])
		}
		if got["eventProperties"] != "1 properties" {
			t.Errorf("eventProperties = %q, want \"1 properties\"", got["eventProperties"])
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Node not found"}`)
		_, err := runWorkflowsNodeGet(cfg(t), "wf_123", "node_missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
