package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var requiredArgRe = regexp.MustCompile(`<([\w-]+)>`)

// match both required `<name>` and optional `[name]` positionals
// - capture group 1 is the required name (empty if optional)
// - capture group 2 is the optional name (empty if required)
var positionalArgRe = regexp.MustCompile(`<([\w-]+)>|\[([\w-]+)\]`)

type Positional struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

// extract every positional argument from a Cobra `Use` string in the
// order they appear. Required args use angle brackets (`<name>`);
// optional args use square brackets (`[name]`).
func parsePositionals(use string) []Positional {
	matches := positionalArgRe.FindAllStringSubmatch(use, -1)
	out := make([]Positional, 0, len(matches))
	for _, m := range matches {
		if m[1] != "" {
			out = append(out, Positional{Name: m[1], Required: true})
		} else if m[2] != "" {
			out = append(out, Positional{Name: m[2]})
		}
	}
	return out
}

func wrapArgsWithNames(cmd *cobra.Command) {
	if cmd.Args != nil {
		v := cmd.Args
		cmd.Args = func(cmd *cobra.Command, args []string) error {
			err := v(cmd, args)
			if err == nil {
				return nil
			}
			required := requiredArgNames(cmd.Use)
			if len(args) < len(required) {
				missing := required[len(args):]
				return fmt.Errorf(`required argument(s) "%s" not set`, strings.Join(missing, `", "`))
			}
			return err
		}
	}
	for _, c := range cmd.Commands() {
		wrapArgsWithNames(c)
	}
}

func requiredArgNames(use string) []string {
	matches := requiredArgRe.FindAllStringSubmatch(use, -1)
	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = m[1]
	}
	return names
}
