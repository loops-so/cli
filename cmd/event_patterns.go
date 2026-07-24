package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func runEventPatternsList(cfg *config.Config, params loops.PaginationParams) ([]loops.EventPatternSummary, error) {
	client := newAPIClient(cfg)
	if params.Cursor != "" {
		patterns, _, err := client.ListEventPatterns(params)
		return patterns, err
	}
	return loops.Paginate(func(cursor string) ([]loops.EventPatternSummary, *loops.Pagination, error) {
		return client.ListEventPatterns(loops.PaginationParams{
			PerPage: params.PerPage,
			Cursor:  cursor,
		})
	})
}

func runEventPatternsGet(cfg *config.Config, id, name string) (*loops.EventPattern, error) {
	client := newAPIClient(cfg)
	if name != "" {
		return client.GetEventPatternByName(name)
	}
	return client.GetEventPattern(id)
}

var eventPatternsCmd = &cobra.Command{
	Use:   "event-patterns",
	Short: "Manage event patterns",
}

var eventPatternsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List event patterns",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		patterns, err := runEventPatternsList(cfg, paginationParams(cmd))
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if patterns == nil {
				patterns = []loops.EventPatternSummary{}
			}
			return printJSON(cmd.OutOrStdout(), patterns)
		}

		if len(patterns) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No event patterns found.")
			return nil
		}

		t := newStyledTable(cmd.OutOrStdout(), "ID", "EVENT NAME", "WEBHOOK PLATFORM")
		for _, p := range patterns {
			t.Row(p.ID, p.EventName, deref(p.IncomingWebhookPlatform))
		}
		return t.Render()
	},
}

var eventPatternsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an event pattern by id or name",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")

		p, err := runEventPatternsGet(cfg, id, name)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), p)
		}

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("id", p.ID)
		t.Row("eventName", p.EventName)
		t.Row("incomingWebhookPlatform", deref(p.IncomingWebhookPlatform))
		t.Row("eventProperties", fmt.Sprintf("%d properties (see -o json)", len(p.EventProperties)))
		return t.Render()
	},
}

func init() {
	addPaginationFlags(eventPatternsListCmd)
	eventPatternsGetCmd.Flags().String("id", "", "Event pattern ID")
	eventPatternsGetCmd.Flags().String("name", "", "Event name")
	eventPatternsGetCmd.MarkFlagsMutuallyExclusive("id", "name")
	eventPatternsGetCmd.MarkFlagsOneRequired("id", "name")
	eventPatternsCmd.AddCommand(eventPatternsListCmd)
	eventPatternsCmd.AddCommand(eventPatternsGetCmd)
	rootCmd.AddCommand(eventPatternsCmd)
}
