package cmd

import (
	"testing"

	"github.com/loops-so/cli/internal/config"
)

func TestRunAuthGet(t *testing.T) {
	t.Run("returns stored key", func(t *testing.T) {
		mockKeyring(t)
		config.Save("key-abc1234", "acme")

		key, err := runAuthGet("acme")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if key != "key-abc1234" {
			t.Errorf("got %q, want %q", key, "key-abc1234")
		}
	})

	t.Run("returns error when key does not exist", func(t *testing.T) {
		mockKeyring(t)

		if _, err := runAuthGet("nonexistent"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when name is empty", func(t *testing.T) {
		mockKeyring(t)

		if _, err := runAuthGet(""); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
