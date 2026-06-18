package cmd

import (
	"net/http"
	"reflect"
	"testing"
)

func TestRunDedicatedSendingIPs(t *testing.T) {
	t.Run("returns assigned ips", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `["192.0.2.1","192.0.2.2"]`)
		ips, err := runDedicatedSendingIPs(cfg(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"192.0.2.1", "192.0.2.2"}
		if !reflect.DeepEqual(ips, want) {
			t.Errorf("got %+v, want %+v", ips, want)
		}
	})

	t.Run("returns empty slice when none assigned", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `[]`)
		ips, err := runDedicatedSendingIPs(cfg(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ips) != 0 {
			t.Errorf("expected empty slice, got %+v", ips)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"error":"Invalid API key"}`)
		_, err := runDedicatedSendingIPs(cfg(t))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
