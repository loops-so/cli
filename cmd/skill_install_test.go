package cmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestSkillInstallArgs(t *testing.T) {
	tests := []struct {
		name   string
		global bool
		yes    bool
		want   []string
	}{
		{
			name: "no flags",
			want: []string{"skills", "add", "https://github.com/loops-so/skills", "--skill", "loops-cli"},
		},
		{
			name:   "global",
			global: true,
			want:   []string{"skills", "add", "https://github.com/loops-so/skills", "--skill", "loops-cli", "--global"},
		},
		{
			name: "yes",
			yes:  true,
			want: []string{"--yes", "skills", "add", "https://github.com/loops-so/skills", "--skill", "loops-cli", "--yes"},
		},
		{
			name:   "global and yes",
			global: true,
			yes:    true,
			want:   []string{"--yes", "skills", "add", "https://github.com/loops-so/skills", "--skill", "loops-cli", "--global", "--yes"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := skillInstallArgs(tc.global, tc.yes)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRunSkillInstallErrorsWhenNpxMissing(t *testing.T) {
	t.Setenv("PATH", "")
	var buf bytes.Buffer
	err := runSkillInstall(&buf, false, false)
	if err == nil {
		t.Fatal("expected error when npx is missing, got nil")
	}
	if !strings.Contains(err.Error(), "npx not found") {
		t.Errorf("expected error to mention 'npx not found', got %v", err)
	}
}
