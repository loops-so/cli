package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

// formatSegmentFilter renders the filter on an audience segment. Unlike a
// campaign's audience-filter, a nil filter on a segment means the reserved
// "all contacts" segment — call that out explicitly.
func formatSegmentFilter(f *loops.AudienceFilter) string {
	if f == nil {
		return "(all contacts)"
	}
	return fmt.Sprintf("match=%s (%d conditions)", f.Match, len(f.Conditions))
}

func runAudienceSegmentsGet(cfg *config.Config, id string) (*loops.AudienceSegment, error) {
	return newAPIClient(cfg).GetAudienceSegment(id)
}

func runAudienceSegmentsList(cfg *config.Config, params loops.PaginationParams) ([]loops.AudienceSegment, error) {
	client := newAPIClient(cfg)
	if params.Cursor != "" {
		segments, _, err := client.ListAudienceSegments(params)
		return segments, err
	}
	return loops.Paginate(func(cursor string) ([]loops.AudienceSegment, *loops.Pagination, error) {
		return client.ListAudienceSegments(loops.PaginationParams{
			PerPage: params.PerPage,
			Cursor:  cursor,
		})
	})
}

var audienceSegmentsCmd = &cobra.Command{
	Use:   "audience-segments",
	Short: "Read audience segments",
}

var audienceSegmentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audience segments",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePickFlags(cmd); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		segments, err := runAudienceSegmentsList(cfg, paginationParams(cmd))
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if segments == nil {
				segments = []loops.AudienceSegment{}
			}
			return printJSON(cmd.OutOrStdout(), segments)
		}

		if len(segments) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No audience segments found.")
			return nil
		}

		headers := []string{"ID", "NAME", "FILTER", "UPDATED"}
		rows := make([][]string, 0, len(segments))
		for _, s := range segments {
			rows = append(rows, []string{
				s.ID,
				s.Name,
				formatSegmentFilter(s.Filter),
				s.UpdatedAt,
			})
		}

		if isPicking(cmd) {
			return runPicker(headers, rows, []pickBinding{
				copyColumnBinding("enter", "copy id", "segment ID", rows, 0, cmd.OutOrStdout()),
			})
		}

		t := newStyledTable(cmd.OutOrStdout(), headers...)
		for _, r := range rows {
			t.Row(r...)
		}
		return t.Render()
	},
}

var audienceSegmentsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get an audience segment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		s, err := runAudienceSegmentsGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), s)
		}

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("segmentId", s.ID)
		t.Row("name", s.Name)
		t.Row("description", deref(s.Description))
		t.Row("filter", formatSegmentFilter(s.Filter))
		t.Row("createdAt", s.CreatedAt)
		t.Row("updatedAt", s.UpdatedAt)
		return t.Render()
	},
}

func init() {
	addPaginationFlags(audienceSegmentsListCmd)
	addPickFlag(audienceSegmentsListCmd)
	audienceSegmentsCmd.AddCommand(audienceSegmentsListCmd)
	audienceSegmentsCmd.AddCommand(audienceSegmentsGetCmd)
	rootCmd.AddCommand(audienceSegmentsCmd)
}
