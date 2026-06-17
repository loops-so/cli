package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func formatCampaignScheduling(s loops.CampaignScheduling) string {
	if s.Method == loops.CampaignSchedulingMethodSchedule && s.Timestamp != nil {
		return "schedule @ " + *s.Timestamp
	}
	return s.Method
}

func runCampaignsGet(cfg *config.Config, id string) (*loops.Campaign, error) {
	return newAPIClient(cfg).GetCampaign(id)
}

func runCampaignsList(cfg *config.Config, params loops.PaginationParams) ([]loops.Campaign, error) {
	client := newAPIClient(cfg)
	if params.Cursor != "" {
		campaigns, _, err := client.ListCampaigns(params)
		return campaigns, err
	}
	return loops.Paginate(func(cursor string) ([]loops.Campaign, *loops.Pagination, error) {
		return client.ListCampaigns(loops.PaginationParams{
			PerPage: params.PerPage,
			Cursor:  cursor,
		})
	})
}

var campaignsCmd = &cobra.Command{
	Use:   "campaigns",
	Short: "Manage campaigns",
}

var campaignsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List campaigns",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePickFlags(cmd); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		campaigns, err := runCampaignsList(cfg, paginationParams(cmd))
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if campaigns == nil {
				campaigns = []loops.Campaign{}
			}
			return printJSON(cmd.OutOrStdout(), campaigns)
		}

		if len(campaigns) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No campaigns found.")
			return nil
		}

		headers := []string{"ID", "MESSAGE ID", "NAME", "STATUS", "SCHEDULING", "UPDATED"}
		rows := make([][]string, 0, len(campaigns))
		for _, c := range campaigns {
			rows = append(rows, []string{
				c.ID,
				deref(c.EmailMessageID),
				c.Name,
				c.Status,
				formatCampaignScheduling(c.Scheduling),
				c.UpdatedAt,
			})
		}

		if isPicking(cmd) {
			out := cmd.OutOrStdout()
			return runPicker(headers, rows, []pickBinding{
				copyColumnBinding("enter", "copy id", "campaign ID", rows, 0, out),
				copyColumnBinding("alt-enter", "copy messageId", "message ID", rows, 1, out),
			})
		}

		t := newStyledTable(cmd.OutOrStdout(), headers...)
		for _, r := range rows {
			t.Row(r...)
		}
		return t.Render()
	},
}

func runCampaignsCreate(cfg *config.Config, req loops.CreateCampaignRequest) (*loops.CampaignCreateResponse, error) {
	return newAPIClient(cfg).CreateCampaign(req)
}

var campaignsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a draft campaign",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		resp, err := runCampaignsCreate(cfg, loops.CreateCampaignRequest{Name: name})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), resp)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created. (id: %s, emailMessageId: %s, contentRevisionId: %s)\n", resp.ID, deref(resp.EmailMessageID), deref(resp.EmailMessageContentRevisionID))
		return nil
	},
}

func runCampaignsUpdate(cfg *config.Config, id string, req loops.UpdateCampaignRequest) (*loops.Campaign, error) {
	return newAPIClient(cfg).UpdateCampaign(id, req)
}

var campaignsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a draft campaign",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		c, err := runCampaignsUpdate(cfg, args[0], loops.UpdateCampaignRequest{
			Name: name,
			Set:  map[string]bool{"name": true},
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), c)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (id: %s)\n\n", c.ID)

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("campaignId", c.ID)
		t.Row("emailMessageId", deref(c.EmailMessageID))
		t.Row("name", c.Name)
		t.Row("status", c.Status)
		t.Row("createdAt", c.CreatedAt)
		t.Row("updatedAt", c.UpdatedAt)
		return t.Render()
	},
}

var campaignsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a campaign",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		c, err := runCampaignsGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), c)
		}

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("campaignId", c.ID)
		t.Row("emailMessageId", deref(c.EmailMessageID))
		t.Row("name", c.Name)
		t.Row("status", c.Status)
		t.Row("createdAt", c.CreatedAt)
		t.Row("updatedAt", c.UpdatedAt)
		return t.Render()
	},
}

func init() {
	addPaginationFlags(campaignsListCmd)
	addPickFlag(campaignsListCmd)
	campaignsCmd.AddCommand(campaignsListCmd)
	campaignsCmd.AddCommand(campaignsGetCmd)

	campaignsCreateCmd.Flags().StringP("name", "n", "", "Campaign name (required)")
	campaignsCreateCmd.MarkFlagRequired("name")
	campaignsCmd.AddCommand(campaignsCreateCmd)

	campaignsUpdateCmd.Flags().StringP("name", "n", "", "Campaign name (required)")
	campaignsUpdateCmd.MarkFlagRequired("name")
	campaignsCmd.AddCommand(campaignsUpdateCmd)

	rootCmd.AddCommand(campaignsCmd)
}
