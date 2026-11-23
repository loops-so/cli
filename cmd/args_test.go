package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestWrapArgsWithNames_LeavesNilArgsAlone(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	bare := &cobra.Command{Use: "bare"}
	root.AddCommand(bare)

	wrapArgsWithNames(root)

	if bare.Args != nil {
		t.Fatal("bare command should not be wrapped")
	}
}

func TestWrapArgsWithNames_PassesThroughOnSuccess(t *testing.T) {
	cmd := &cobra.Command{Use: "login <name>", Args: cobra.ExactArgs(1)}
	wrapArgsWithNames(cmd)

	if err := cmd.Args(cmd, []string{"x"}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestWrapArgsWithNames_NamesMissingArg(t *testing.T) {
	cmd := &cobra.Command{Use: "login <name>", Args: cobra.ExactArgs(1)}
	wrapArgsWithNames(cmd)

	err := cmd.Args(cmd, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := err.Error(), `required argument(s) "name" not set`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestWrapArgsWithNames_NamesMultipleMissingArgs(t *testing.T) {
	cmd := &cobra.Command{Use: "thing <a> <b>", Args: cobra.ExactArgs(2)}
	wrapArgsWithNames(cmd)

	err := cmd.Args(cmd, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := err.Error(), `required argument(s) "a", "b" not set`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestWrapArgsWithNames_NamesOnlyTrailingMissingArgs(t *testing.T) {
	cmd := &cobra.Command{Use: "thing <a> <b>", Args: cobra.ExactArgs(2)}
	wrapArgsWithNames(cmd)

	err := cmd.Args(cmd, []string{"first"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := err.Error(), `required argument(s) "b" not set`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestWrapArgsWithNames_FallsBackOnTooManyArgs(t *testing.T) {
	cmd := &cobra.Command{Use: "login <name>", Args: cobra.ExactArgs(1)}
	wrapArgsWithNames(cmd)

	err := cmd.Args(cmd, []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := err.Error(), "accepts 1 arg(s), received 2"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestWrapArgsWithNames_RecursesIntoChildren(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	parent := &cobra.Command{Use: "parent"}
	grandchild := &cobra.Command{Use: "grandchild <event>", Args: cobra.ExactArgs(1)}
	parent.AddCommand(grandchild)
	root.AddCommand(parent)

	wrapArgsWithNames(root)

	err := grandchild.Args(grandchild, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := err.Error(), `required argument(s) "event" not set`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestRequiredArgNames(t *testing.T) {
	cases := []struct {
		use  string
		want []string
	}{
		{"login <name>", []string{"name"}},
		{"thing <a> <b>", []string{"a", "b"}},
		{"use [name]", nil},
		{"mixed <id> [extra]", []string{"id"}},
		{"get <workflow-id> <node-id>", []string{"workflow-id", "node-id"}},
		{"bare", nil},
	}
	for _, tc := range cases {
		got := requiredArgNames(tc.use)
		if !reflect.DeepEqual(got, tc.want) && !(len(got) == 0 && len(tc.want) == 0) {
			t.Errorf("requiredArgNames(%q) = %v, want %v", tc.use, got, tc.want)
		}
	}
}
