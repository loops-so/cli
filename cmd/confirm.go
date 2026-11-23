package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

var stdinIsTerminal = func() bool {
	return term.IsTerminal(os.Stdin.Fd())
}

func addYesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt")
}

// confirmAction returns nil when --yes is set or the user answers yes to
// prompt. A prompt is not possible with --output json or when stdin is not a
// terminal; --yes is then required.
func confirmAction(cmd *cobra.Command, prompt string) error {
	if yes, _ := cmd.Flags().GetBool("yes"); yes {
		return nil
	}
	if isJSONOutput() || !stdinIsTerminal() {
		return errors.New("confirmation required: re-run with --yes")
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N]: ", prompt)
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return nil
	}
	return errors.New("cancelled")
}
