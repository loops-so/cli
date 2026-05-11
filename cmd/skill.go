package cmd

import "github.com/spf13/cobra"

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage the Loops CLI skill for AI agents",
}

func init() {
	rootCmd.AddCommand(skillCmd)
}
