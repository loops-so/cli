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
