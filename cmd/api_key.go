package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func runAPIKey(cfg *config.Config) (*loops.APIKeyResponse, error) {
	return newAPIClient(cfg).GetAPIKey()
}

var apiKeyCmd = &cobra.Command{
	Use:   "api-key",
	Short: "Validate your API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		result, err := runAPIKey(cfg)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), result)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Valid API key for team: %s\n", result.TeamName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(apiKeyCmd)
}
