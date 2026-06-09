package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

const pickerHeaderLines = 2

func addPickFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("pick", false, "Interactively pick a row with fzf")
}

func isPicking(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("pick")
	return v
}

// reject --pick combined with --output json
func validatePickFlags(cmd *cobra.Command) error {
	if isPicking(cmd) && isJSONOutput() {
		return errors.New("--pick cannot be combined with --output json")
	}
	return nil
}

// pickBinding is a single key → action mapping inside the picker. the first
// binding passed to runPicker is the default (key must be "enter").
type pickBinding struct {
	Key    string
	Label  string
	Action func(rowIdx int) error
}

// render the styled table for headers/rows and prefix each data line
// with "<idx>\t" so fzf can identify the original row regardless of
// filtering/reordering. header lines pass through unchanged.
func buildPickerInput(headers []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	t := newStyledTable(&buf, headers...)
	for _, r := range rows {
		t.Row(r...)
	}
	if err := t.Render(); err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) <= pickerHeaderLines {
		return nil, errors.New("no rows to pick")
	}

	var out bytes.Buffer
	for i := range pickerHeaderLines {
		out.WriteByte('\t')
		out.WriteString(lines[i])
		out.WriteByte('\n')
	}
	for idx, line := range lines[pickerHeaderLines:] {
		fmt.Fprintf(&out, "%d\t%s\n", idx, line)
	}
	return out.Bytes(), nil
}

// parse a single fzf row line of "<rowIdx>\t<rendered_row>" and validate idx.
func parsePickerSelection(s string, numRows int) (int, error) {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0, errors.New("empty selection")
	}
	prefix, _, ok := strings.Cut(s, "\t")
	if !ok {
		return 0, errors.New("unexpected selection format")
	}
	idx, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("invalid row index: %w", err)
	}
	if idx < 0 || idx >= numRows {
		return 0, fmt.Errorf("row index %d out of range [0, %d)", idx, numRows)
	}
	return idx, nil
}

// parse fzf --expect output: first line is the pressed key (empty for the
// default Enter), second line is the row prefixed by buildPickerInput.
func parsePickerOutput(s string, numRows int) (key string, rowIdx int, err error) {
	s = strings.TrimRight(s, "\n")
	keyLine, rowLine, ok := strings.Cut(s, "\n")
	if !ok {
		return "", 0, errors.New("unexpected fzf output format")
	}
	rowIdx, err = parsePickerSelection(rowLine, numRows)
	if err != nil {
		return "", 0, err
	}
	return keyLine, rowIdx, nil
}

// build an fzf --color spec that matches the CLI's fang color scheme:
// the table column header color is reused for accents (prompt, pointer,
// matches, etc.) and the comment color for info text. base scheme
// follows the detected terminal background.
func pickerColorSpec() string {
	cs := fangColorScheme()
	base := "dark"
	if !isDarkBackground() {
		base = "light"
	}
	accent := hexColor(cs.Title)
	dim := hexColor(cs.Comment)
	return fmt.Sprintf(
		"%s,header:%s,prompt:%s,pointer:%s,marker:%s,spinner:%s,hl:%s,hl+:%s,info:%s",
		base, accent, accent, accent, accent, accent, accent, accent, dim,
	)
}

func renderPickerHeader(bindings []pickBinding) string {
	parts := make([]string, len(bindings))
	for i, b := range bindings {
		parts[i] = fmt.Sprintf(" %s ▶ %s", b.Key, b.Label)
	}
	return strings.Join(parts, "\n")
}

func runPicker(headers []string, rows [][]string, bindings []pickBinding) error {
	if len(bindings) == 0 {
		return errors.New("runPicker: at least one binding required")
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		return errors.New("--pick requires fzf to be installed and on PATH")
	}

	input, err := buildPickerInput(headers, rows)
	if err != nil {
		return err
	}

	args := []string{
		"--ansi",
		"--layout", "reverse-list",
		"--header-lines", strconv.Itoa(pickerHeaderLines),
		"--delimiter", "\t",
		"--with-nth", "2..",
		"--header", renderPickerHeader(bindings),
		"--header-first",
		"--color", pickerColorSpec(),
	}

	expect := make([]string, 0, len(bindings)-1)
	for _, b := range bindings[1:] {
		expect = append(expect, b.Key)
	}
	if len(expect) > 0 {
		args = append(args, "--expect", strings.Join(expect, ","))
	}

	fzf := exec.Command("fzf", args...)
	fzf.Stdin = bytes.NewReader(input)
	fzf.Stderr = os.Stderr
	selBytes, err := fzf.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			switch exitErr.ExitCode() {
			case 1, 130:
				return nil
			}
		}
		return fmt.Errorf("fzf: %w", err)
	}

	output := string(selBytes)
	if strings.TrimRight(output, "\n") == "" {
		return nil
	}

	var key string
	var rowIdx int
	if len(expect) == 0 {
		rowIdx, err = parsePickerSelection(output, len(rows))
	} else {
		key, rowIdx, err = parsePickerOutput(output, len(rows))
	}
	if err != nil {
		return err
	}

	for _, b := range bindings {
		if (key == "" && b.Key == "enter") || b.Key == key {
			return b.Action(rowIdx)
		}
	}
	return fmt.Errorf("no binding for key %q", key)
}

func copyColumnBinding(key, headerLabel, copyLabel string, rows [][]string, col int, out io.Writer) pickBinding {
	return pickBinding{
		Key:   key,
		Label: headerLabel,
		Action: func(rowIdx int) error {
			v := rows[rowIdx][col]
			if err := clipboard.WriteAll(v); err != nil {
				return fmt.Errorf("failed to copy to clipboard: %w", err)
			}
			fmt.Fprintf(out, "Copied %s: %s\n", copyLabel, v)
			return nil
		},
	}
}
