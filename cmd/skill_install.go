package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

const (
	skillsRepoURL = "https://github.com/loops-so/skills"
	skillName     = "loops-cli"
)

var (
	skillInstallGlobal bool
	skillInstallYes    bool
)

func skillInstallArgs(global, yes bool) []string {
	args := []string{"skills", "add", skillsRepoURL, "--skill", skillName}
	if global {
		args = append(args, "--global")
	}
	if yes {
		args = append(args, "--yes")
	}
	return args
}

func runSkillInstall(stderr io.Writer, global, yes bool) error {
	if _, err := exec.LookPath("npx"); err != nil {
		return fmt.Errorf("npx not found on PATH")
	}

	args := skillInstallArgs(global, yes)
	fmt.Fprintf(stderr, "Running: npx %s\n", strings.Join(args, " "))

	c := exec.Command("npx", args...)
	c.Stdin = os.Stdin
	c.Stdout = stderr
	c.Stderr = stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("npx: %w", err)
	}
	return nil
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Loops CLI skill via npx",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runSkillInstall(os.Stderr, skillInstallGlobal, skillInstallYes); err != nil {
			return err
		}
		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), Result{Success: true, Message: "skill installed"})
		}
		return nil
	},
}

func init() {
	skillInstallCmd.Flags().BoolVarP(&skillInstallGlobal, "global", "g", false, "Install the skill globally")
	skillInstallCmd.Flags().BoolVarP(&skillInstallYes, "yes", "y", false, "Skip confirmation prompts")
	skillCmd.AddCommand(skillInstallCmd)
}
