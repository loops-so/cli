package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

const sampleGroupBody = `{
	"id": "grp_abc",
	"name": "Onboarding",
	"description": "Welcome flow",
	"createdAt": "2026-04-01T10:00:00Z",
	"updatedAt": "2026-04-20T10:00:00Z"
}`

func TestRunCampaignGroupsList(t *testing.T) {
	t.Run("returns groups", func(t *testing.T) {
		body := `{"pagination":{"nextCursor":""},"data":[
			{"id":"grp_1","name":"Onboarding","description":"","createdAt":"2026-04-01","updatedAt":"2026-04-02"},
			{"id":"grp_2","name":"Promotions","description":"Sales","createdAt":"2026-03-01","updatedAt":"2026-03-05"}
		]}`
		got := serveJSONCapture(t, http.StatusOK, body)

		groups, err := runCampaignGroupsList(cfg(t), loops.PaginationParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Path != "/campaign-groups" {
			t.Errorf("Path = %q, want /campaign-groups", got.Path)
		}
		if len(groups) != 2 {
			t.Fatalf("len = %d, want 2", len(groups))
		}
		if groups[0].ID != "grp_1" || groups[1].ID != "grp_2" {
			t.Errorf("ids = %q, %q", groups[0].ID, groups[1].ID)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"error":"unauthorized"}`)
		_, err := runCampaignGroupsList(cfg(t), loops.PaginationParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunCampaignGroupsGet(t *testing.T) {
	got := serveJSONCapture(t, http.StatusOK, sampleGroupBody)
	g, err := runCampaignGroupsGet(cfg(t), "grp_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != "/campaign-groups/grp_abc" {
		t.Errorf("Path = %q", got.Path)
	}
	if g.ID != "grp_abc" || g.Name != "Onboarding" {
		t.Errorf("group = %+v", g)
	}
}

func TestRunCampaignGroupsCreate(t *testing.T) {
	got := serveJSONCapture(t, http.StatusCreated, sampleGroupBody)
	_, err := runCampaignGroupsCreate(cfg(t), loops.CreateGroupRequest{
		Name:        "Onboarding",
		Description: "Welcome flow",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != "/campaign-groups" || got.Method != http.MethodPost {
		t.Errorf("Path/Method = %q/%q", got.Path, got.Method)
	}

	var sent map[string]any
	if err := json.Unmarshal(got.Body, &sent); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if sent["name"] != "Onboarding" {
		t.Errorf("name = %v", sent["name"])
	}
	if sent["description"] != "Welcome flow" {
		t.Errorf("description = %v", sent["description"])
	}
}

func TestRunCampaignGroupsUpdate(t *testing.T) {
	got := serveJSONCapture(t, http.StatusOK, sampleGroupBody)
	_, err := runCampaignGroupsUpdate(cfg(t), "grp_abc", loops.UpdateGroupRequest{
		Name: "Renamed",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != "/campaign-groups/grp_abc" || got.Method != http.MethodPost {
		t.Errorf("Path/Method = %q/%q", got.Path, got.Method)
	}

	var sent map[string]any
	if err := json.Unmarshal(got.Body, &sent); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if sent["name"] != "Renamed" {
		t.Errorf("name = %v", sent["name"])
	}
	if _, ok := sent["description"]; ok {
		t.Errorf("description should be omitted when empty, got %v", sent)
	}
}

// The transactional-groups path uses the same helper machinery; spot-check
// each verb's endpoint to confirm the dispatch is correct.

func TestRunTransactionalGroupsEndpoints(t *testing.T) {
	t.Run("list hits /transactional-groups", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, `{"pagination":{"nextCursor":""},"data":[]}`)
		if _, err := runTransactionalGroupsList(cfg(t), loops.PaginationParams{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Path != "/transactional-groups" {
			t.Errorf("Path = %q", got.Path)
		}
	})

	t.Run("get hits /transactional-groups/:id", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, sampleGroupBody)
		if _, err := runTransactionalGroupsGet(cfg(t), "grp_abc"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Path != "/transactional-groups/grp_abc" {
			t.Errorf("Path = %q", got.Path)
		}
	})

	t.Run("create hits /transactional-groups", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, sampleGroupBody)
		if _, err := runTransactionalGroupsCreate(cfg(t), loops.CreateGroupRequest{Name: "Onboarding"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Path != "/transactional-groups" || got.Method != http.MethodPost {
			t.Errorf("Path/Method = %q/%q", got.Path, got.Method)
		}
	})

	t.Run("update hits /transactional-groups/:id", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, sampleGroupBody)
		if _, err := runTransactionalGroupsUpdate(cfg(t), "grp_abc", loops.UpdateGroupRequest{Name: "Renamed"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Path != "/transactional-groups/grp_abc" || got.Method != http.MethodPost {
			t.Errorf("Path/Method = %q/%q", got.Path, got.Method)
		}
	})
}
