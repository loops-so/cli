package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func stubStdinTerminal(t *testing.T, isTerminal bool) {
	t.Helper()
	orig := stdinIsTerminal
	stdinIsTerminal = func() bool { return isTerminal }
	t.Cleanup(func() { stdinIsTerminal = orig })
}

func confirmCmd(t *testing.T, input string, args ...string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{}
	addYesFlag(cmd)
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	stderr := &bytes.Buffer{}
	cmd.SetIn(strings.NewReader(input))
	cmd.SetErr(stderr)
	cmd.SetOut(io.Discard)
	return cmd, stderr
}

func TestConfirmAction(t *testing.T) {
	useOutputFormat(t, "text")

	t.Run("--yes skips the prompt", func(t *testing.T) {
		stubStdinTerminal(t, false)
		cmd, stderr := confirmCmd(t, "", "--yes")
		if err := confirmAction(cmd, "Proceed?"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stderr.Len() != 0 {
			t.Errorf("prompt written with --yes: %q", stderr.String())
		}
	})

	t.Run("requires --yes when stdin is not a terminal", func(t *testing.T) {
		stubStdinTerminal(t, false)
		cmd, _ := confirmCmd(t, "y\n")
		if err := confirmAction(cmd, "Proceed?"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("requires --yes with json output", func(t *testing.T) {
		stubStdinTerminal(t, true)
		useOutputFormat(t, "json")

		cmd, _ := confirmCmd(t, "y\n")
		if err := confirmAction(cmd, "Proceed?"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	for _, answer := range []string{"y\n", "Y\n", "yes\n", " yes \n"} {
		t.Run("accepts "+strings.TrimSpace(answer), func(t *testing.T) {
			stubStdinTerminal(t, true)
			cmd, stderr := confirmCmd(t, answer)
			if err := confirmAction(cmd, "Proceed?"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := stderr.String(); got != "Proceed? [y/N]: " {
				t.Errorf("prompt = %q", got)
			}
		})
	}

	for _, answer := range []string{"\n", "n\n", "nope\n", ""} {
		t.Run("declines on "+strings.TrimSpace(answer), func(t *testing.T) {
			stubStdinTerminal(t, true)
			cmd, _ := confirmCmd(t, answer)
			if err := confirmAction(cmd, "Proceed?"); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
