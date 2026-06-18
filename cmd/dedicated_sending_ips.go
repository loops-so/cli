package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/spf13/cobra"
)

func runDedicatedSendingIPs(cfg *config.Config) ([]string, error) {
	return newAPIClient(cfg).GetDedicatedSendingIPs()
}

var dedicatedSendingIPsCmd = &cobra.Command{
	Use:   "dedicated-sending-ips",
	Short: "List dedicated sending IP addresses assigned to your team",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		ips, err := runDedicatedSendingIPs(cfg)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if ips == nil {
				ips = []string{}
			}
			return printJSON(cmd.OutOrStdout(), ips)
		}

		if len(ips) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No dedicated sending IPs assigned.")
			return nil
		}

		for _, ip := range ips {
			fmt.Fprintln(cmd.OutOrStdout(), ip)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dedicatedSendingIPsCmd)
}
