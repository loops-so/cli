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
	skillInstallAll    bool
)

func skillInstallArgs(global, yes, all bool) []string {
	args := []string{}
	// --yes for npx (--all implies non-interactive intent)
	if yes || all {
		args = append(args, "--yes")
	}
	args = append(args, "skills", "add", skillsRepoURL)
	if all {
		args = append(args, "--all")
	} else {
		args = append(args, "--skill", skillName)
	}
	if global {
		args = append(args, "--global")
	}
	// --yes for skills (--all already implies -y to skills)
	if yes && !all {
		args = append(args, "--yes")
	}
	return args
}

func runSkillInstall(stderr io.Writer, global, yes, all bool) error {
	if _, err := exec.LookPath("npx"); err != nil {
		return fmt.Errorf("npx not found on PATH")
	}

	args := skillInstallArgs(global, yes, all)
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
	Short: "Install the Loops CLI skill via 'skills add'",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runSkillInstall(os.Stderr, skillInstallGlobal, skillInstallYes, skillInstallAll); err != nil {
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
	skillInstallCmd.Flags().BoolVarP(&skillInstallAll, "all", "a", false, "Install all Loops skills (not just the CLI skill)")
	skillCmd.AddCommand(skillInstallCmd)
}
