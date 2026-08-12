package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

// newCreateNodeFlagsCmd builds a command with the `workflows nodes create`
// flags registered and set from the given map, for testing
// parseCreateWorkflowNodeFlags in isolation.
func newCreateNodeFlagsCmd(t *testing.T, flags map[string]string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{}
	for _, name := range []string{"node-type", "insert-mode", "from-node-id", "to-node-id", "before-node-id", "expected-revision-id"} {
		c.Flags().String(name, "", "")
	}
	for k, v := range flags {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("set --%s: %v", k, err)
		}
	}
	return c
}

func TestParseCreateWorkflowNodeFlags(t *testing.T) {
	t.Run("between sets from/to node ids", func(t *testing.T) {
		req, err := parseCreateWorkflowNodeFlags(newCreateNodeFlagsCmd(t, map[string]string{
			"node-type":    loops.CreateWorkflowNodeTypeTimerAction,
			"insert-mode":  loops.WorkflowInsertModeBetween,
			"from-node-id": "n1",
			"to-node-id":   "n2",
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.FromNodeID != "n1" || req.ToNodeID != "n2" {
			t.Errorf("from/to = %q/%q, want n1/n2", req.FromNodeID, req.ToNodeID)
		}
		if req.BeforeNodeID != "" {
			t.Errorf("BeforeNodeID = %q, want empty", req.BeforeNodeID)
		}
	})

	t.Run("before maps before-node-id to ToNodeID (not BeforeNodeID)", func(t *testing.T) {
		req, err := parseCreateWorkflowNodeFlags(newCreateNodeFlagsCmd(t, map[string]string{
			"node-type":      loops.CreateWorkflowNodeTypeTimerAction,
			"insert-mode":    loops.WorkflowInsertModeBefore,
			"before-node-id": "n3",
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.ToNodeID != "n3" {
			t.Errorf("ToNodeID = %q, want n3", req.ToNodeID)
		}
		if req.BeforeNodeID != "" {
			t.Errorf("BeforeNodeID = %q, want empty (deprecated field must not be sent)", req.BeforeNodeID)
		}
		if req.FromNodeID != "" {
			t.Errorf("FromNodeID = %q, want empty", req.FromNodeID)
		}
	})

	t.Run("after sets FromNodeID", func(t *testing.T) {
		req, err := parseCreateWorkflowNodeFlags(newCreateNodeFlagsCmd(t, map[string]string{
			"node-type":    loops.CreateWorkflowNodeTypeTimerAction,
			"insert-mode":  loops.WorkflowInsertModeAfter,
			"from-node-id": "n1",
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.InsertMode != loops.WorkflowInsertModeAfter {
			t.Errorf("InsertMode = %q, want after", req.InsertMode)
		}
		if req.FromNodeID != "n1" {
			t.Errorf("FromNodeID = %q, want n1", req.FromNodeID)
		}
		if req.ToNodeID != "" || req.BeforeNodeID != "" {
			t.Errorf("to/before = %q/%q, want empty", req.ToNodeID, req.BeforeNodeID)
		}
	})

	t.Run("expected-revision-id is passed through when set", func(t *testing.T) {
		req, err := parseCreateWorkflowNodeFlags(newCreateNodeFlagsCmd(t, map[string]string{
			"node-type":            loops.CreateWorkflowNodeTypeTimerAction,
			"insert-mode":          loops.WorkflowInsertModeAfter,
			"from-node-id":         "n1",
			"expected-revision-id": "rev_1",
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.ExpectedRevisionID == nil || *req.ExpectedRevisionID != "rev_1" {
			t.Errorf("ExpectedRevisionID = %v, want rev_1", req.ExpectedRevisionID)
		}
	})

	errCases := []struct {
		name  string
		flags map[string]string
	}{
		{"unknown node-type", map[string]string{"node-type": "Nonsense", "insert-mode": loops.WorkflowInsertModeAfter, "from-node-id": "n1"}},
		{"unknown insert-mode", map[string]string{"node-type": loops.CreateWorkflowNodeTypeTimerAction, "insert-mode": "sideways", "from-node-id": "n1"}},
		{"between missing to", map[string]string{"node-type": loops.CreateWorkflowNodeTypeTimerAction, "insert-mode": loops.WorkflowInsertModeBetween, "from-node-id": "n1"}},
		{"before missing before-node-id", map[string]string{"node-type": loops.CreateWorkflowNodeTypeTimerAction, "insert-mode": loops.WorkflowInsertModeBefore}},
		{"after missing from-node-id", map[string]string{"node-type": loops.CreateWorkflowNodeTypeTimerAction, "insert-mode": loops.WorkflowInsertModeAfter}},
	}
	for _, tc := range errCases {
		t.Run(tc.name+" is an error", func(t *testing.T) {
			if _, err := parseCreateWorkflowNodeFlags(newCreateNodeFlagsCmd(t, tc.flags)); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestRunWorkflowsNodeCreate(t *testing.T) {
	body := `{
		"node": {
			"typeName": "TimerAction",
			"id": "node_new",
			"nextNodeIds": ["n2"],
			"amount": 0,
			"unit": "m",
			"workflowRevisionId": "rev_2"
		},
		"workflow": {
			"id": "wf_1",
			"name": "WF",
			"status": "Draft",
			"workflowRevisionId": "rev_2",
			"mailingListId": null,
			"rootNodeId": null,
			"nodes": {}
		}
	}`

	t.Run("between mode sends from/to node ids", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		resp, err := runWorkflowsNodeCreate(cfg(t), "wf_1", loops.CreateWorkflowNodeRequest{
			InsertMode:   loops.WorkflowInsertModeBetween,
			NodeTypeName: loops.CreateWorkflowNodeTypeTimerAction,
			FromNodeID:   "n1",
			ToNodeID:     "n2",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Node.TypeName != loops.WorkflowNodeTypeTimerAction {
			t.Errorf("Node.TypeName = %q, want TimerAction", resp.Node.TypeName)
		}
		if resp.Workflow.ID != "wf_1" {
			t.Errorf("Workflow.ID = %q, want wf_1", resp.Workflow.ID)
		}
		if cap.Path != "/workflows/wf_1/nodes" {
			t.Errorf("Path = %q, want /workflows/wf_1/nodes", cap.Path)
		}

		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v\nraw: %s", err, cap.Body)
		}
		if sent["insertMode"] != "between" {
			t.Errorf("insertMode = %v, want between", sent["insertMode"])
		}
		if sent["nodeTypeName"] != "TimerAction" {
			t.Errorf("nodeTypeName = %v, want TimerAction", sent["nodeTypeName"])
		}
		if sent["fromNodeId"] != "n1" || sent["toNodeId"] != "n2" {
			t.Errorf("from/to = %v/%v, want n1/n2", sent["fromNodeId"], sent["toNodeId"])
		}
		if _, ok := sent["beforeNodeId"]; ok {
			t.Error("beforeNodeId present; want omitted for between mode")
		}
	})

	t.Run("before mode sends beforeNodeId", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		_, err := runWorkflowsNodeCreate(cfg(t), "wf_1", loops.CreateWorkflowNodeRequest{
			InsertMode:   loops.WorkflowInsertModeBefore,
			NodeTypeName: loops.CreateWorkflowNodeTypeTimerAction,
			BeforeNodeID: "n3",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if sent["beforeNodeId"] != "n3" {
			t.Errorf("beforeNodeId = %v, want n3", sent["beforeNodeId"])
		}
		if _, ok := sent["fromNodeId"]; ok {
			t.Error("fromNodeId present; want omitted for before mode")
		}
	})
}

func TestParseUpdateWorkflowNodePayload(t *testing.T) {
	writePayload := func(t *testing.T, contents string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "payload.json")
		if err := os.WriteFile(p, []byte(contents), 0o600); err != nil {
			t.Fatalf("write payload: %v", err)
		}
		return p
	}

	t.Run("timer action config payload", func(t *testing.T) {
		path := writePayload(t, `{"typeName":"TimerAction","amount":3,"unit":"d"}`)
		payload, err := parseUpdateWorkflowNodePayload(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.TypeName != loops.WorkflowNodeTypeTimerAction {
			t.Errorf("TypeName = %q, want TimerAction", payload.TypeName)
		}
		if payload.TimerAction == nil {
			t.Fatal("TimerAction = nil, want populated")
		}
		if payload.TimerAction.Amount == nil || *payload.TimerAction.Amount != 3 {
			t.Errorf("Amount = %v, want 3", payload.TimerAction.Amount)
		}
		if payload.TimerAction.Unit != loops.WorkflowTimerUnitDays {
			t.Errorf("Unit = %q, want d", payload.TimerAction.Unit)
		}

		// Round-trips with the SDK MarshalJSON (config variant, no typeName).
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if got := string(raw); got != `{"amount":3,"unit":"d"}` {
			t.Errorf("marshaled = %s", got)
		}
	})

	t.Run("event trigger payload carries typeName", func(t *testing.T) {
		path := writePayload(t, `{"typeName":"EventTrigger","eventName":"signup","reEligible":true}`)
		payload, err := parseUpdateWorkflowNodePayload(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.EventTrigger == nil {
			t.Fatal("EventTrigger = nil, want populated")
		}
		if deref(payload.EventTrigger.EventName) != "signup" {
			t.Errorf("EventName = %q, want signup", deref(payload.EventTrigger.EventName))
		}
		raw, _ := json.Marshal(payload)
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal marshaled: %v", err)
		}
		if m["typeName"] != "EventTrigger" {
			t.Errorf("typeName = %v, want EventTrigger", m["typeName"])
		}
	})

	t.Run("missing typeName is an error", func(t *testing.T) {
		path := writePayload(t, `{"amount":3}`)
		if _, err := parseUpdateWorkflowNodePayload(path); err == nil {
			t.Error("expected error for missing typeName")
		}
	})

	t.Run("unknown typeName is an error", func(t *testing.T) {
		path := writePayload(t, `{"typeName":"Nonsense"}`)
		if _, err := parseUpdateWorkflowNodePayload(path); err == nil {
			t.Error("expected error for unknown typeName")
		}
	})

	t.Run("missing file is an error", func(t *testing.T) {
		if _, err := parseUpdateWorkflowNodePayload(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Error("expected error for missing file")
		}
	})
}

func TestRunWorkflowsNodeUpdate(t *testing.T) {
	body := `{
		"typeName": "TimerAction",
		"id": "node_t",
		"nextNodeIds": ["n2"],
		"amount": 3,
		"unit": "d",
		"workflowRevisionId": "rev_5"
	}`

	t.Run("sends payload and expectedRevisionId", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		amount := 3.0
		rev := "rev_4"
		node, err := runWorkflowsNodeUpdate(cfg(t), "wf_1", "node_t", loops.UpdateWorkflowNodeRequest{
			ExpectedRevisionID: &rev,
			Payload: loops.UpdateWorkflowNodePayload{
				TypeName:    loops.WorkflowNodeTypeTimerAction,
				TimerAction: &loops.WorkflowTimerActionPayload{Amount: &amount, Unit: loops.WorkflowTimerUnitDays},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if node.WorkflowRevisionID != "rev_5" {
			t.Errorf("WorkflowRevisionID = %q, want rev_5", node.WorkflowRevisionID)
		}
		if cap.Path != "/workflows/wf_1/nodes/node_t" {
			t.Errorf("Path = %q, want /workflows/wf_1/nodes/node_t", cap.Path)
		}

		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v\nraw: %s", err, cap.Body)
		}
		if sent["expectedRevisionId"] != "rev_4" {
			t.Errorf("expectedRevisionId = %v, want rev_4", sent["expectedRevisionId"])
		}
		payload, ok := sent["payload"].(map[string]any)
		if !ok {
			t.Fatalf("payload not an object: %v", sent["payload"])
		}
		if payload["amount"] != 3.0 || payload["unit"] != "d" {
			t.Errorf("payload = %v", payload)
		}
	})
}

func TestRunWorkflowsNodeAddBranch(t *testing.T) {
	body := `{
		"node": {
			"typeName": "BranchNode",
			"id": "node_b",
			"nextNodeIds": ["a","b"],
			"workflowRevisionId": "rev_6"
		},
		"workflow": {
			"id": "wf_1",
			"name": "WF",
			"status": "Draft",
			"workflowRevisionId": "rev_6",
			"mailingListId": null,
			"rootNodeId": null,
			"nodes": {}
		}
	}`

	t.Run("posts to add-branch path", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		resp, err := runWorkflowsNodeAddBranch(cfg(t), "wf_1", "node_b", loops.AddWorkflowBranchRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Node.WorkflowRevisionID != "rev_6" {
			t.Errorf("revision = %q, want rev_6", resp.Node.WorkflowRevisionID)
		}
		if cap.Path != "/workflows/wf_1/nodes/node_b/add-branch" {
			t.Errorf("Path = %q, want .../add-branch", cap.Path)
		}
	})
}

func TestRunWorkflowsNodeReroute(t *testing.T) {
	body := `{
		"typeName": "TimerAction",
		"id": "node_r",
		"nextNodeIds": ["n2"],
		"amount": 0,
		"unit": "m",
		"workflowRevisionId": "rev_8",
		"workflow": {
			"id": "wf_1",
			"name": "WF",
			"status": "Draft",
			"workflowRevisionId": "rev_8",
			"mailingListId": null,
			"rootNodeId": null,
			"nodes": {}
		}
	}`

	t.Run("posts to reroute path with new target and revision", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		rev := "rev_7"
		resp, err := runWorkflowsNodeReroute(cfg(t), "wf_1", "node_r", loops.RerouteNodeConnectionRequest{
			ExpectedRevisionID: &rev,
			NewTargetNodeID:    "node_new",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.WorkflowRevisionID != "rev_8" {
			t.Errorf("WorkflowRevisionID = %q, want rev_8", resp.WorkflowRevisionID)
		}
		if resp.Workflow.ID != "wf_1" {
			t.Errorf("Workflow.ID = %q, want wf_1", resp.Workflow.ID)
		}
		if cap.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", cap.Method)
		}
		if cap.Path != "/workflows/wf_1/nodes/node_r/reroute" {
			t.Errorf("Path = %q, want /workflows/wf_1/nodes/node_r/reroute", cap.Path)
		}

		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v\nraw: %s", err, cap.Body)
		}
		if sent["newTargetNodeId"] != "node_new" {
			t.Errorf("newTargetNodeId = %v, want node_new", sent["newTargetNodeId"])
		}
		if sent["expectedRevisionId"] != "rev_7" {
			t.Errorf("expectedRevisionId = %v, want rev_7", sent["expectedRevisionId"])
		}
	})
}

func TestRunWorkflowsNodeDelete(t *testing.T) {
	body := `{"status":"deleted","nodeIds":["node_x"],"workflowRevisionId":"rev_7","queuedContactCount":0}`

	t.Run("non-recursive hits the node path", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		r, err := runWorkflowsNodeDelete(cfg(t), "wf_1", "node_x", false, loops.DeleteWorkflowNodeRequest{
			DryRun:              true,
			QueuedContactPolicy: loops.WorkflowQueuedContactPolicyFail,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Status != "deleted" {
			t.Errorf("Status = %q, want deleted", r.Status)
		}
		if cap.Method != http.MethodDelete {
			t.Errorf("Method = %q, want DELETE", cap.Method)
		}
		if cap.Path != "/workflows/wf_1/nodes/node_x" {
			t.Errorf("Path = %q, want /workflows/wf_1/nodes/node_x", cap.Path)
		}
		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if sent["dryRun"] != true {
			t.Errorf("dryRun = %v, want true", sent["dryRun"])
		}
		if sent["queuedContactPolicy"] != "fail" {
			t.Errorf("queuedContactPolicy = %v, want fail", sent["queuedContactPolicy"])
		}
	})

	t.Run("recursive hits the recursive path", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		_, err := runWorkflowsNodeDelete(cfg(t), "wf_1", "node_x", true, loops.DeleteWorkflowNodeRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cap.Path != "/workflows/wf_1/nodes/node_x/recursive" {
			t.Errorf("Path = %q, want .../recursive", cap.Path)
		}
	})
}
