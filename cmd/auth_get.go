package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/spf13/cobra"
)

var authGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Print a stored API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		key, err := runAuthGet(name)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), struct {
				Name   string `json:"name"`
				APIKey string `json:"apiKey"`
			}{name, key})
		}

		fmt.Fprintln(cmd.OutOrStdout(), key)
		return nil
	},
}

func runAuthGet(name string) (string, error) {
	return config.Get(name)
}

func init() {
	authCmd.AddCommand(authGetCmd)
}
