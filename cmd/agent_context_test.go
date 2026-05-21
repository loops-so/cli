package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestBuildAgentContext_TopLevel(t *testing.T) {
	ctx := buildAgentContext(rootCmd)

	if ctx.Name != "loops" {
		t.Errorf("name = %q, want %q", ctx.Name, "loops")
	}
	if ctx.Short == "" {
		t.Error("short should not be empty")
	}
	if ctx.Long == "" {
		t.Error("long should not be empty")
	}

	names := agentFlagNames(ctx.GlobalFlags)
	for _, want := range []string{"debug", "output", "team"} {
		if !agentSliceContains(names, want) {
			t.Errorf("globalFlags missing %q (got %v)", want, names)
		}
	}
}

func TestBuildAgentContext_TreeInvariants(t *testing.T) {
	ctx := buildAgentContext(rootCmd)
	walkAgentCommands(ctx.Commands, func(c Command) {
		if c.Name == "" {
			t.Errorf("empty name at path %v", c.Path)
		}
		if len(c.Path) == 0 {
			t.Errorf("empty path for command %q", c.Name)
		} else if c.Path[len(c.Path)-1] != c.Name {
			t.Errorf("path %v doesn't end with name %q", c.Path, c.Name)
		}
		if c.Name == "help" || c.Name == "completion" {
			t.Errorf("auto-injected command leaked: %v", c.Path)
		}
		if findAgentFlag(c.Flags, "help") != nil {
			t.Errorf("--help flag leaked into %v", c.Path)
		}
	})
}

func TestBuildAgentContext_Deterministic(t *testing.T) {
	a, err := json.Marshal(buildAgentContext(rootCmd))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(buildAgentContext(rootCmd))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("output is not byte-deterministic across calls")
	}
}

func TestBuildAgentContext_ContactsCreate(t *testing.T) {
	ctx := buildAgentContext(rootCmd)
	create := findAgentCommand(ctx.Commands, "contacts", "create")
	if create == nil {
		t.Fatal("contacts create not found")
	}
	email := findAgentFlag(create.Flags, "email")
	if email == nil {
		t.Fatal("contacts create --email not found")
	}
	if !email.Required {
		t.Error("contacts create --email should be required")
	}
}

func TestBuildAgentContext_DepthThree(t *testing.T) {
	ctx := buildAgentContext(rootCmd)
	check := findAgentCommand(ctx.Commands, "contacts", "suppression", "check")
	if check == nil {
		t.Fatal("contacts suppression check not found")
	}
	if findAgentFlag(check.Flags, "email") == nil {
		t.Error("contacts suppression check missing --email")
	}
	if findAgentFlag(check.Flags, "user-id") == nil {
		t.Error("contacts suppression check missing --user-id")
	}
}

func TestBuildAgentContext_FlagGroups(t *testing.T) {
	ctx := buildAgentContext(rootCmd)
	upd := findAgentCommand(ctx.Commands, "email-messages", "update")
	if upd == nil {
		t.Fatal("email-messages update not found")
	}
	if !groupContaining(upd.FlagGroups.MutuallyExclusive, "lmx", "lmx-file") {
		t.Errorf("email-messages update missing mutex group covering lmx + lmx-file; got %v",
			upd.FlagGroups.MutuallyExclusive)
	}
	if !groupContaining(upd.FlagGroups.OneRequired, "lmx") {
		t.Errorf("email-messages update missing oneRequired group covering content fields; got %v",
			upd.FlagGroups.OneRequired)
	}
}

func TestBuildAgentContext_NoRootPersistentLeak(t *testing.T) {
	ctx := buildAgentContext(rootCmd)
	rootPersistent := []string{"output", "team", "debug"}
	walkAgentCommands(ctx.Commands, func(c Command) {
		for _, name := range rootPersistent {
			if findAgentFlag(c.Flags, name) != nil {
				t.Errorf("root persistent flag %q leaked into %v", name, c.Path)
			}
		}
	})
}

func TestExtractFlagGroups_AllAnnotationKeys(t *testing.T) {
	cmd := &cobra.Command{Use: "synth"}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		cmd.Flags().String(name, "", "")
	}
	cmd.MarkFlagsOneRequired("a", "b")
	cmd.MarkFlagsMutuallyExclusive("c", "d")
	cmd.MarkFlagsRequiredTogether("e", "f")

	groups := extractFlagGroups(cmd)
	if !groupListContains(groups.OneRequired, []string{"a", "b"}) {
		t.Errorf("oneRequired missing [a b]; got %v", groups.OneRequired)
	}
	if !groupListContains(groups.MutuallyExclusive, []string{"c", "d"}) {
		t.Errorf("mutuallyExclusive missing [c d]; got %v", groups.MutuallyExclusive)
	}
	if !groupListContains(groups.RequiredTogether, []string{"e", "f"}) {
		t.Errorf("requiredTogether missing [e f]; got %v", groups.RequiredTogether)
	}
}

func TestBuildAgentContext_HiddenCommands(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	root.AddCommand(&cobra.Command{Use: "secret", Hidden: true})

	ctx := buildAgentContext(root)
	secret := findAgentCommand(ctx.Commands, "secret")
	if secret == nil {
		t.Fatal("secret not found in output (hidden commands should still be included)")
	}
	if !secret.Hidden {
		t.Error("secret should be hidden=true")
	}
}

func TestBuildAgentContext_PositionalArgs(t *testing.T) {
	ctx := buildAgentContext(rootCmd)

	login := findAgentCommand(ctx.Commands, "auth", "login")
	if login == nil {
		t.Fatal("auth login not found")
	}
	if len(login.Args) != 1 {
		t.Fatalf("auth login should have 1 positional, got %d", len(login.Args))
	}
	if login.Args[0].Name != "name" || !login.Args[0].Required {
		t.Errorf("auth login positional = %+v, want {name required}", login.Args[0])
	}

	use := findAgentCommand(ctx.Commands, "auth", "use")
	if use == nil {
		t.Fatal("auth use not found")
	}
	if len(use.Args) != 1 || use.Args[0].Required {
		t.Errorf("auth use should have 1 optional positional, got %+v", use.Args)
	}
}

func TestBuildAgentContext_SelfPresent(t *testing.T) {
	ctx := buildAgentContext(rootCmd)
	self := findAgentCommand(ctx.Commands, "agent-context")
	if self == nil {
		t.Fatal("agent-context not in own output")
	}
	if self.Hidden {
		t.Error("agent-context should not be hidden")
	}
}

func TestAgentContextCmd_RunEmitsValidJSON(t *testing.T) {
	var buf bytes.Buffer
	agentContextCmd.SetOut(&buf)
	t.Cleanup(func() { agentContextCmd.SetOut(nil) })

	if err := agentContextCmd.RunE(agentContextCmd, nil); err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if v["name"] != "loops" {
		t.Errorf("output name = %v, want loops", v["name"])
	}
}

func TestParsePositionals(t *testing.T) {
	cases := []struct {
		use  string
		want []Positional
	}{
		{"login <name>", []Positional{{Name: "name", Required: true}}},
		{"use [name]", []Positional{{Name: "name"}}},
		{"thing <a> <b>", []Positional{{Name: "a", Required: true}, {Name: "b", Required: true}}},
		{"mixed <id> [extra]", []Positional{{Name: "id", Required: true}, {Name: "extra"}}},
		{"bare", nil},
	}
	for _, tc := range cases {
		got := parsePositionals(tc.use)
		if len(got) != len(tc.want) {
			t.Errorf("parsePositionals(%q) = %+v, want %+v", tc.use, got, tc.want)
			continue
		}
		for i, p := range got {
			if p != tc.want[i] {
				t.Errorf("parsePositionals(%q)[%d] = %+v, want %+v", tc.use, i, p, tc.want[i])
			}
		}
	}
}

func walkAgentCommands(cmds []Command, fn func(Command)) {
	for _, c := range cmds {
		fn(c)
		walkAgentCommands(c.Commands, fn)
	}
}

func findAgentCommand(cmds []Command, path ...string) *Command {
	if len(path) == 0 {
		return nil
	}
	for i := range cmds {
		if cmds[i].Name == path[0] {
			if len(path) == 1 {
				return &cmds[i]
			}
			return findAgentCommand(cmds[i].Commands, path[1:]...)
		}
	}
	return nil
}

func findAgentFlag(flags []Flag, name string) *Flag {
	for i := range flags {
		if flags[i].Name == name {
			return &flags[i]
		}
	}
	return nil
}

func agentFlagNames(flags []Flag) []string {
	out := make([]string, len(flags))
	for i, f := range flags {
		out[i] = f.Name
	}
	return out
}

func agentSliceContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func groupListContains(groups [][]string, want []string) bool {
	wantStr := strings.Join(want, " ")
	for _, g := range groups {
		if strings.Join(g, " ") == wantStr {
			return true
		}
	}
	return false
}

func groupContaining(groups [][]string, want ...string) bool {
	for _, g := range groups {
		have := make(map[string]bool, len(g))
		for _, name := range g {
			have[name] = true
		}
		ok := true
		for _, name := range want {
			if !have[name] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
