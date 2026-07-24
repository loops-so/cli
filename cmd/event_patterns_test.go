package cmd

import (
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunEventPatternsList(t *testing.T) {
	t.Run("returns event patterns", func(t *testing.T) {
		body := `{
			"pagination":{"nextCursor":""},
			"data":[
				{"id":"ep_1","eventName":"purchase.completed","incomingWebhookPlatform":"stripe"},
				{"id":"ep_2","eventName":"signed_up","incomingWebhookPlatform":null}
			]
		}`
		cap := serveJSONCapture(t, http.StatusOK, body)

		patterns, err := runEventPatternsList(cfg(t), loops.PaginationParams{PerPage: "10"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(patterns) != 2 {
			t.Fatalf("expected 2 patterns, got %d", len(patterns))
		}
		if patterns[0].ID != "ep_1" {
			t.Errorf("ID = %q, want ep_1", patterns[0].ID)
		}
		if deref(patterns[0].IncomingWebhookPlatform) != "stripe" {
			t.Errorf("IncomingWebhookPlatform = %q, want stripe", deref(patterns[0].IncomingWebhookPlatform))
		}
		if patterns[1].IncomingWebhookPlatform != nil {
			t.Errorf("IncomingWebhookPlatform = %v, want nil", patterns[1].IncomingWebhookPlatform)
		}
		if cap.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", cap.Method)
		}
		if cap.Path != "/event-patterns?perPage=10" {
			t.Errorf("Path = %q, want /event-patterns?perPage=10", cap.Path)
		}
	})

	t.Run("single page when cursor set", func(t *testing.T) {
		body := `{"pagination":{"nextCursor":"next"},"data":[{"id":"ep_1","eventName":"purchase.completed","incomingWebhookPlatform":null}]}`
		cap := serveJSONCapture(t, http.StatusOK, body)

		patterns, err := runEventPatternsList(cfg(t), loops.PaginationParams{Cursor: "abc"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(patterns) != 1 {
			t.Fatalf("expected 1 pattern, got %d", len(patterns))
		}
		if cap.Path != "/event-patterns?cursor=abc" {
			t.Errorf("Path = %q, want /event-patterns?cursor=abc", cap.Path)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"error":"unauthorized"}`)
		_, err := runEventPatternsList(cfg(t), loops.PaginationParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunEventPatternsGet(t *testing.T) {
	body := `{
		"id": "ep_123",
		"eventName": "purchase.completed",
		"incomingWebhookPlatform": "stripe",
		"eventProperties": [
			{"name": "amount", "type": "number"},
			{"name": "currency", "type": "string"}
		]
	}`

	t.Run("by id", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		p, err := runEventPatternsGet(cfg(t), "ep_123", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ID != "ep_123" {
			t.Errorf("ID = %q, want ep_123", p.ID)
		}
		if p.EventName != "purchase.completed" {
			t.Errorf("EventName = %q, want purchase.completed", p.EventName)
		}
		if deref(p.IncomingWebhookPlatform) != "stripe" {
			t.Errorf("IncomingWebhookPlatform = %q, want stripe", deref(p.IncomingWebhookPlatform))
		}
		if len(p.EventProperties) != 2 {
			t.Fatalf("expected 2 event properties, got %d", len(p.EventProperties))
		}
		if p.EventProperties[0].Name != "amount" || p.EventProperties[0].Type != "number" {
			t.Errorf("EventProperties[0] = %+v, want {amount number}", p.EventProperties[0])
		}
		if cap.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", cap.Method)
		}
		if cap.Path != "/event-patterns/ep_123" {
			t.Errorf("Path = %q, want /event-patterns/ep_123", cap.Path)
		}
	})

	t.Run("by name", func(t *testing.T) {
		cap := serveJSONCapture(t, http.StatusOK, body)
		p, err := runEventPatternsGet(cfg(t), "", "purchase.completed")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ID != "ep_123" {
			t.Errorf("ID = %q, want ep_123", p.ID)
		}
		if cap.Path != "/event-patterns/by-name/purchase.completed" {
			t.Errorf("Path = %q, want /event-patterns/by-name/purchase.completed", cap.Path)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Event pattern not found"}`)
		_, err := runEventPatternsGet(cfg(t), "ep_missing", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
