package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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

		return printAudienceSegment(cmd, s)
	},
}

func printAudienceSegment(cmd *cobra.Command, s *loops.AudienceSegment) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("segmentId", s.ID)
	t.Row("name", s.Name)
	t.Row("description", deref(s.Description))
	t.Row("filter", formatSegmentFilter(s.Filter))
	t.Row("createdAt", s.CreatedAt)
	t.Row("updatedAt", s.UpdatedAt)
	return t.Render()
}

func runAudienceSegmentsCreate(cfg *config.Config, req loops.CreateAudienceSegmentRequest) (*loops.AudienceSegment, error) {
	return newAPIClient(cfg).CreateAudienceSegment(req)
}

// filterFromCmd resolves the audience filter from either the inline --filter
// JSON string or the --filter-file path (exactly one, enforced as a flag
// group). Selection is value-based so it stays correct when RunE is called
// directly in tests, which bypasses cobra's flag-group validation.
func filterFromCmd(cmd *cobra.Command) (loops.AudienceFilter, error) {
	inline, _ := cmd.Flags().GetString("filter")
	path, _ := cmd.Flags().GetString("filter-file")

	var data []byte
	src := "--filter"
	switch {
	case inline != "" && path != "":
		return loops.AudienceFilter{}, fmt.Errorf("--filter and --filter-file are mutually exclusive")
	case inline != "":
		data = []byte(inline)
	case path != "":
		src = "--filter-file"
		b, err := os.ReadFile(path)
		if err != nil {
			return loops.AudienceFilter{}, fmt.Errorf("read --filter-file: %w", err)
		}
		data = b
	default:
		return loops.AudienceFilter{}, fmt.Errorf("one of --filter or --filter-file is required")
	}

	var filter loops.AudienceFilter
	if err := json.Unmarshal(data, &filter); err != nil {
		return loops.AudienceFilter{}, fmt.Errorf("parse %s: %w", src, err)
	}
	return filter, nil
}

var audienceSegmentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an audience segment",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")

		filter, err := filterFromCmd(cmd)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		s, err := runAudienceSegmentsCreate(cfg, loops.CreateAudienceSegmentRequest{
			Name:        name,
			Description: description,
			Filter:      filter,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), s)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created. (id: %s)\n\n", s.ID)
		return printAudienceSegment(cmd, s)
	},
}

func init() {
	addPaginationFlags(audienceSegmentsListCmd)
	addPickFlag(audienceSegmentsListCmd)
	audienceSegmentsCmd.AddCommand(audienceSegmentsListCmd)
	audienceSegmentsCmd.AddCommand(audienceSegmentsGetCmd)

	audienceSegmentsCreateCmd.Flags().StringP("name", "n", "", "Segment name")
	audienceSegmentsCreateCmd.Flags().String("description", "", "Segment description")
	audienceSegmentsCreateCmd.Flags().String("filter", "", "Audience filter as an inline JSON string")
	audienceSegmentsCreateCmd.Flags().String("filter-file", "", "Path to a JSON file with the audience filter")
	audienceSegmentsCreateCmd.MarkFlagRequired("name")
	audienceSegmentsCreateCmd.MarkFlagsMutuallyExclusive("filter", "filter-file")
	audienceSegmentsCreateCmd.MarkFlagsOneRequired("filter", "filter-file")
	audienceSegmentsCmd.AddCommand(audienceSegmentsCreateCmd)

	rootCmd.AddCommand(audienceSegmentsCmd)
}
