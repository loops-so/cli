package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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

func formatAudienceFilter(f *loops.AudienceFilter) string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("match=%s (%d conditions)", f.Match, len(f.Conditions))
}

// campaignFieldParams holds the targeting/scheduling fields shared by
// `campaigns create` and `campaigns update`. Set records which fields the
// user explicitly provided (keyed by JSON field name) so partial updates can
// send only those fields.
//
// For nullable fields (MailingListID, AudienceSegmentID, AudienceFilter), a
// pointer of nil with the corresponding Set key true encodes "send null" —
// which the API treats as a clear. CLI users opt in via the string sentinel
// "null".
type campaignFieldParams struct {
	Name              string
	CampaignGroupID   string
	MailingListID     *string
	AudienceSegmentID *string
	AudienceFilter    *loops.AudienceFilter
	Scheduling        *loops.CampaignSchedulingRequest
	Set               map[string]bool
}

// nullSentinel is the value users pass to a nullable string flag to clear the
// server-side field (sent as JSON null). Empty string is rejected so users
// don't accidentally write empty values.
const nullSentinel = "null"

func addCampaignFieldFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "Campaign name")
	cmd.Flags().String("campaign-group-id", "", "Campaign group ID")
	cmd.Flags().String("mailing-list-id", "", `Mailing list ID to target. Pass "null" to clear.`)
	cmd.Flags().String("audience-segment-id", "", `Audience segment ID to target. Pass "null" to clear.`)
	cmd.Flags().String("audience-filter-file", "", `Path to a JSON file with an ad-hoc audience filter. Pass "null" to clear.`)
	cmd.Flags().Bool("schedule-now", false, "Send immediately when published")
	cmd.Flags().String("schedule-at", "", "Send at the given ISO 8601 timestamp (e.g. 2026-07-01T12:00:00Z)")
	cmd.MarkFlagsMutuallyExclusive("audience-segment-id", "audience-filter-file")
	cmd.MarkFlagsMutuallyExclusive("schedule-now", "schedule-at")
}

// readNullableFlag returns (pointerOrNil, set, err) for a string flag that
// supports the "null" sentinel. Empty string is rejected.
func readNullableFlag(cmd *cobra.Command, flagName string) (*string, bool, error) {
	if !cmd.Flags().Changed(flagName) {
		return nil, false, nil
	}
	v, _ := cmd.Flags().GetString(flagName)
	if v == "" {
		return nil, false, fmt.Errorf(`--%s requires a value; pass "null" to clear`, flagName)
	}
	if v == nullSentinel {
		return nil, true, nil
	}
	return &v, true, nil
}

func campaignFieldParamsFromCmd(cmd *cobra.Command) (campaignFieldParams, error) {
	p := campaignFieldParams{Set: map[string]bool{}}

	if cmd.Flags().Changed("name") {
		p.Name, _ = cmd.Flags().GetString("name")
		p.Set["name"] = true
	}
	if cmd.Flags().Changed("campaign-group-id") {
		p.CampaignGroupID, _ = cmd.Flags().GetString("campaign-group-id")
		p.Set["campaignGroupId"] = true
	}
	if v, set, err := readNullableFlag(cmd, "mailing-list-id"); err != nil {
		return p, err
	} else if set {
		p.MailingListID = v
		p.Set["mailingListId"] = true
	}
	if v, set, err := readNullableFlag(cmd, "audience-segment-id"); err != nil {
		return p, err
	} else if set {
		p.AudienceSegmentID = v
		p.Set["audienceSegmentId"] = true
	}
	if cmd.Flags().Changed("audience-filter-file") {
		path, _ := cmd.Flags().GetString("audience-filter-file")
		if path == "" {
			return p, fmt.Errorf(`--audience-filter-file requires a value; pass "null" to clear`)
		}
		if path == nullSentinel {
			p.AudienceFilter = nil
			p.Set["audienceFilter"] = true
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				return p, fmt.Errorf("read --audience-filter-file: %w", err)
			}
			var f loops.AudienceFilter
			if err := json.Unmarshal(data, &f); err != nil {
				return p, fmt.Errorf("parse --audience-filter-file: %w", err)
			}
			p.AudienceFilter = &f
			p.Set["audienceFilter"] = true
		}
	}
	if cmd.Flags().Changed("schedule-now") {
		p.Scheduling = &loops.CampaignSchedulingRequest{Method: loops.CampaignSchedulingMethodNow}
		p.Set["scheduling"] = true
	}
	if cmd.Flags().Changed("schedule-at") {
		ts, _ := cmd.Flags().GetString("schedule-at")
		p.Scheduling = &loops.CampaignSchedulingRequest{
			Method:    loops.CampaignSchedulingMethodSchedule,
			Timestamp: ts,
		}
		p.Set["scheduling"] = true
	}
	return p, nil
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
		params, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		resp, err := runCampaignsCreate(cfg, loops.CreateCampaignRequest{
			Name:              params.Name,
			CampaignGroupID:   params.CampaignGroupID,
			MailingListID:     params.MailingListID,
			AudienceSegmentID: params.AudienceSegmentID,
			AudienceFilter:    params.AudienceFilter,
			Scheduling:        params.Scheduling,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), resp)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created. (id: %s, emailMessageId: %s, contentRevisionId: %s)\n\n", resp.ID, deref(resp.EmailMessageID), deref(resp.EmailMessageContentRevisionID))
		return printCampaign(cmd, &resp.Campaign)
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
		params, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		c, err := runCampaignsUpdate(cfg, args[0], loops.UpdateCampaignRequest{
			Name:              params.Name,
			CampaignGroupID:   params.CampaignGroupID,
			MailingListID:     params.MailingListID,
			AudienceSegmentID: params.AudienceSegmentID,
			AudienceFilter:    params.AudienceFilter,
			Scheduling:        params.Scheduling,
			Set:               params.Set,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), c)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (id: %s)\n\n", c.ID)
		return printCampaign(cmd, c)
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

		return printCampaign(cmd, c)
	},
}

func printCampaign(cmd *cobra.Command, c *loops.Campaign) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("campaignId", c.ID)
	t.Row("emailMessageId", deref(c.EmailMessageID))
	t.Row("name", c.Name)
	t.Row("status", c.Status)
	t.Row("campaignGroupId", deref(c.CampaignGroupID))
	t.Row("mailingListId", deref(c.MailingListID))
	t.Row("audienceSegmentId", deref(c.AudienceSegmentID))
	t.Row("audienceFilter", formatAudienceFilter(c.AudienceFilter))
	t.Row("scheduling", formatCampaignScheduling(c.Scheduling))
	t.Row("createdAt", c.CreatedAt)
	t.Row("updatedAt", c.UpdatedAt)
	return t.Render()
}

func init() {
	addPaginationFlags(campaignsListCmd)
	addPickFlag(campaignsListCmd)
	campaignsCmd.AddCommand(campaignsListCmd)
	campaignsCmd.AddCommand(campaignsGetCmd)

	addCampaignFieldFlags(campaignsCreateCmd)
	campaignsCreateCmd.MarkFlagRequired("name")
	campaignsCmd.AddCommand(campaignsCreateCmd)

	addCampaignFieldFlags(campaignsUpdateCmd)
	campaignsUpdateCmd.MarkFlagsOneRequired(
		"name",
		"campaign-group-id",
		"mailing-list-id",
		"audience-segment-id",
		"audience-filter-file",
		"schedule-now",
		"schedule-at",
	)
	campaignsCmd.AddCommand(campaignsUpdateCmd)

	rootCmd.AddCommand(campaignsCmd)
}
