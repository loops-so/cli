package cmd

import (
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("per-page", 0, "Results per page (10-50, default 20)")
	cmd.Flags().String("cursor", "", "Pagination cursor for a specific page")
}

func paginationParams(cmd *cobra.Command) loops.PaginationParams {
	perPage, _ := cmd.Flags().GetInt("per-page")
	cursor, _ := cmd.Flags().GetString("cursor")
	return loops.PaginationParams{
		PerPage: perPage,
		Cursor:  cursor,
	}
}
