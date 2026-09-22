package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func stubVersionArtPick(t *testing.T, idx int) {
	t.Helper()
	orig := pickVersionArt
	pickVersionArt = func(int) int { return idx }
	t.Cleanup(func() { pickVersionArt = orig })
}

func TestVersionArt_EveryEntryRenders(t *testing.T) {
	for i, raw := range versionArts {
		stubVersionArtPick(t, i)
		got := versionArt()
		want := strings.TrimPrefix(raw, "\n")
		if got != want {
			t.Errorf("entry %d: got %q, want %q", i, got, want)
		}
		if strings.HasPrefix(got, "\n") {
			t.Errorf("entry %d: leading newline not trimmed", i)
		}
		if strings.TrimSpace(got) == "" {
			t.Errorf("entry %d: art is empty", i)
		}
		for n, line := range strings.Split(got, "\n") {
			if strings.TrimRight(line, " \t") != line {
				t.Errorf("entry %d line %d: trailing whitespace", i, n+1)
			}
		}
	}
}

func TestVersionArt_DefaultPickerStaysInRange(t *testing.T) {
	n := len(versionArts)
	for range 1000 {
		if got := pickVersionArt(n); got < 0 || got >= n {
			t.Fatalf("pick %d out of range [0,%d)", got, n)
		}
	}
}

func TestVersionFlag_PrintsSelectedArt(t *testing.T) {
	stubVersionArtPick(t, 0)

	origVersion := rootCmd.Version
	rootCmd.Version = "1.2.3"
	t.Cleanup(func() { rootCmd.Version = origVersion })

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}

	got := out.String()
	art := strings.TrimPrefix(versionArts[0], "\n")
	if !strings.HasPrefix(got, art) {
		t.Errorf("output does not start with selected art:\n%s", got)
	}
	if !strings.Contains(got, "loops version 1.2.3") {
		t.Errorf("output missing version line:\n%s", got)
	}
}
