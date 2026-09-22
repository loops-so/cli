package cmd

import (
	"fmt"
	"strings"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print the resolved configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, keyResp, pc, err := runAuthStatus()
		if err != nil {
			return err
		}

		masked := maskKey(cfg.APIKey)

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), struct {
				ActiveKey   string `json:"activeKey"`
				APIKey      string `json:"apiKey"`
				EndpointURL string `json:"endpointUrl"`
				TeamName    string `json:"teamName"`
			}{pc.ActiveTeam, masked, cfg.EndpointURL, keyResp.TeamName})
		}

		activeKey := pc.ActiveTeam
		if activeKey == "" {
			activeKey = "(none)"
		}

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("activeKey", activeKey)
		t.Row("apiKey", masked)
		t.Row("teamName", keyResp.TeamName)
		t.Row("endpointUrl", cfg.EndpointURL)
		return t.Render()
	},
}

func runAuthStatus() (*config.Config, *loops.APIKeyResponse, *config.PersistentConfig, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	keyResp, err := newAPIClient(cfg).GetAPIKey()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("API key verification failed: %w", err)
	}
	pc, err := config.LoadPersistentConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	return cfg, keyResp, pc, nil
}

func maskKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return fmt.Sprintf("%s%s", strings.Repeat("*", len(key)-4), key[len(key)-4:])
}

func init() {
	authCmd.AddCommand(statusCmd)
}
