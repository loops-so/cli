package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loops-so/loops-go"
)

func TestRunEmailMessagesRender(t *testing.T) {
	t.Run("sends variables and returns html", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, `{"html":"<html><title>Hi Pat</title></html>"}`)
		resp, err := runEmailMessagesRender(cfg(t), "em_abc123", loops.EmailMessageRenderRequest{
			ContactProperties: map[string]string{"firstName": "Pat"},
			EventProperties:   map[string]string{"signupSource": "homepage"},
			DataVariables:     map[string]string{"orderId": "123"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.HTML != "<html><title>Hi Pat</title></html>" {
			t.Errorf("HTML = %q", resp.HTML)
		}
		if got.Path != "/email-messages/em_abc123/render" {
			t.Errorf("Path = %q", got.Path)
		}
		if got.Method != http.MethodPost {
			t.Errorf("Method = %q", got.Method)
		}

		var sent map[string]map[string]any
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if sent["contactProperties"]["firstName"] != "Pat" {
			t.Errorf("contactProperties.firstName = %v", sent["contactProperties"]["firstName"])
		}
		if sent["eventProperties"]["signupSource"] != "homepage" {
			t.Errorf("eventProperties.signupSource = %v", sent["eventProperties"]["signupSource"])
		}
		if sent["dataVariables"]["orderId"] != "123" {
			t.Errorf("dataVariables.orderId = %v", sent["dataVariables"]["orderId"])
		}
	})

	t.Run("omits empty variables", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, `{"html":"<html></html>"}`)
		if _, err := runEmailMessagesRender(cfg(t), "em_abc123", loops.EmailMessageRenderRequest{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got.Body) != "{}" {
			t.Errorf("Body = %s, want {}", got.Body)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		serveJSON(t, http.StatusBadRequest, `{"success":false,"message":"bad"}`)
		if _, err := runEmailMessagesRender(cfg(t), "em_abc123", loops.EmailMessageRenderRequest{}); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
