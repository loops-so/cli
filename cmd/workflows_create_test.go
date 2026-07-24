package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunWorkflowsCreate(t *testing.T) {
	body := `{
		"id": "wf_new",
		"name": "Onboarding",
		"description": "New user series",
		"status": "Draft",
		"workflowRevisionId": "rev_1",
		"mailingListId": "ml_1",
		"rootNodeId": null,
		"nodes": {}
	}`

	t.Run("returns workflow on success", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		ml := "ml_1"
		w, err := runWorkflowsCreate(cfg(t), loops.CreateWorkflowRequest{
			Name:          "Onboarding",
			Description:   "New user series",
			MailingListID: &ml,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.ID != "wf_new" {
			t.Errorf("ID = %q, want wf_new", w.ID)
		}

		if cap.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", cap.Method)
		}
		if cap.Path != "/workflows" {
			t.Errorf("Path = %q, want /workflows", cap.Path)
		}

		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v\nraw: %s", err, cap.Body)
		}
		if sent["name"] != "Onboarding" {
			t.Errorf("name = %v, want Onboarding", sent["name"])
		}
		if sent["description"] != "New user series" {
			t.Errorf("description = %v", sent["description"])
		}
		if sent["mailingListId"] != "ml_1" {
			t.Errorf("mailingListId = %v, want ml_1", sent["mailingListId"])
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"success":false,"message":"name is required"}`)
		_, err := runWorkflowsCreate(cfg(t), loops.CreateWorkflowRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunWorkflowsUpdate(t *testing.T) {
	body := `{
		"id": "wf_1",
		"name": "Renamed",
		"status": "Draft",
		"workflowRevisionId": "rev_2",
		"mailingListId": null,
		"rootNodeId": null,
		"nodes": {}
	}`

	t.Run("sends expectedRevisionId and fields", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		rev := "rev_1"
		w, err := runWorkflowsUpdate(cfg(t), "wf_1", loops.UpdateWorkflowPropertiesRequest{
			ExpectedRevisionID: &rev,
			Name:               "Renamed",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.Name != "Renamed" {
			t.Errorf("Name = %q, want Renamed", w.Name)
		}
		if cap.Path != "/workflows/wf_1" {
			t.Errorf("Path = %q, want /workflows/wf_1", cap.Path)
		}

		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v\nraw: %s", err, cap.Body)
		}
		if sent["expectedRevisionId"] != "rev_1" {
			t.Errorf("expectedRevisionId = %v, want rev_1", sent["expectedRevisionId"])
		}
		if sent["name"] != "Renamed" {
			t.Errorf("name = %v, want Renamed", sent["name"])
		}
	})

	t.Run("sends null expectedRevisionId when unset", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		_, err := runWorkflowsUpdate(cfg(t), "wf_1", loops.UpdateWorkflowPropertiesRequest{
			Name: "Renamed",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := sent["expectedRevisionId"]; !ok {
			t.Error("expectedRevisionId key missing; want present (null)")
		}
		if sent["expectedRevisionId"] != nil {
			t.Errorf("expectedRevisionId = %v, want null", sent["expectedRevisionId"])
		}
	})
}

func TestRunWorkflowsChangeMailingList(t *testing.T) {
	t.Run("sends mailingListId, dryRun, and policy", func(t *testing.T) {
		body := `{"status":"updated","mailingListId":"ml_2","workflowRevisionId":"rev_3","queuedContactCount":0}`
		cap := serveJSONCapture(t, http.StatusOK, body)
		rev := "rev_2"
		ml := "ml_2"
		r, err := runWorkflowsChangeMailingList(cfg(t), "wf_1", loops.ChangeWorkflowMailingListRequest{
			ExpectedRevisionID:  &rev,
			MailingListID:       &ml,
			DryRun:              true,
			QueuedContactPolicy: loops.WorkflowQueuedContactPolicyDiscard,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Status != "updated" {
			t.Errorf("Status = %q, want updated", r.Status)
		}
		if cap.Path != "/workflows/wf_1/mailing-list" {
			t.Errorf("Path = %q, want /workflows/wf_1/mailing-list", cap.Path)
		}

		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v\nraw: %s", err, cap.Body)
		}
		if sent["mailingListId"] != "ml_2" {
			t.Errorf("mailingListId = %v, want ml_2", sent["mailingListId"])
		}
		if sent["dryRun"] != true {
			t.Errorf("dryRun = %v, want true", sent["dryRun"])
		}
		if sent["queuedContactPolicy"] != "discard" {
			t.Errorf("queuedContactPolicy = %v, want discard", sent["queuedContactPolicy"])
		}
	})

	t.Run("sends null mailingListId to clear", func(t *testing.T) {
		body := `{"status":"updated","mailingListId":null,"workflowRevisionId":"rev_3","queuedContactCount":0}`
		cap := serveJSONCapture(t, http.StatusOK, body)
		_, err := runWorkflowsChangeMailingList(cfg(t), "wf_1", loops.ChangeWorkflowMailingListRequest{
			MailingListID: nil,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var sent map[string]any
		if err := json.Unmarshal(cap.Body, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := sent["mailingListId"]; !ok {
			t.Error("mailingListId key missing; want present (null)")
		}
		if sent["mailingListId"] != nil {
			t.Errorf("mailingListId = %v, want null", sent["mailingListId"])
		}
	})
}

func TestValidateQueuedContactPolicy(t *testing.T) {
	for _, v := range []string{"", "fail", "discard"} {
		if err := validateQueuedContactPolicy(v); err != nil {
			t.Errorf("validateQueuedContactPolicy(%q) = %v, want nil", v, err)
		}
	}
	if err := validateQueuedContactPolicy("bogus"); err == nil {
		t.Error("validateQueuedContactPolicy(bogus) = nil, want error")
	}
}
