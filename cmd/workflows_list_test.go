package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunWorkflowsList(t *testing.T) {
	t.Run("returns workflows", func(t *testing.T) {
		body := `{
			"pagination":{"nextCursor":""},
			"data":[
				{"id":"wf_1","name":"Welcome","createdAt":"2026-04-01","updatedAt":"2026-04-02"},
				{"id":"wf_2","name":"Onboarding","createdAt":"2026-05-01","updatedAt":"2026-05-02"}
			]
		}`
		cap := serveJSONCapture(t, http.StatusOK, body)

		workflows, err := runWorkflowsList(cfg(t), loops.PaginationParams{PerPage: "10"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(workflows) != 2 {
			t.Fatalf("expected 2 workflows, got %d", len(workflows))
		}
		if workflows[0].ID != "wf_1" {
			t.Errorf("ID = %q, want wf_1", workflows[0].ID)
		}
		if workflows[1].Name != "Onboarding" {
			t.Errorf("Name = %q, want Onboarding", workflows[1].Name)
		}
		if cap.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", cap.Method)
		}
		if cap.Path != "/workflows?perPage=10" {
			t.Errorf("Path = %q, want /workflows?perPage=10", cap.Path)
		}
	})

	t.Run("single page when cursor set", func(t *testing.T) {
		body := `{"pagination":{"nextCursor":"next"},"data":[{"id":"wf_1","name":"Welcome","createdAt":"","updatedAt":""}]}`
		cap := serveJSONCapture(t, http.StatusOK, body)

		workflows, err := runWorkflowsList(cfg(t), loops.PaginationParams{Cursor: "abc"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(workflows) != 1 {
			t.Fatalf("expected 1 workflow, got %d", len(workflows))
		}
		if cap.Path != "/workflows?cursor=abc" {
			t.Errorf("Path = %q, want /workflows?cursor=abc", cap.Path)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"error":"unauthorized"}`)
		_, err := runWorkflowsList(cfg(t), loops.PaginationParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
