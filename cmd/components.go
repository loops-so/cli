package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func runComponentsGet(cfg *config.Config, id string) (*loops.Component, error) {
	return newAPIClient(cfg).GetComponent(id)
}

func runComponentsList(cfg *config.Config, params loops.PaginationParams) ([]loops.Component, error) {
	client := newAPIClient(cfg)
	if params.Cursor != "" {
		components, _, err := client.ListComponents(params)
		return components, err
	}
	return loops.Paginate(func(cursor string) ([]loops.Component, *loops.Pagination, error) {
		return client.ListComponents(loops.PaginationParams{
			PerPage: params.PerPage,
			Cursor:  cursor,
		})
	})
}

var componentsCmd = &cobra.Command{
	Use:   "components",
	Short: "Manage components",
}

var componentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List components",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePickFlags(cmd); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		components, err := runComponentsList(cfg, paginationParams(cmd))
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if components == nil {
				components = []loops.Component{}
			}
			return printJSON(cmd.OutOrStdout(), components)
		}

		if len(components) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No components found.")
			return nil
		}

		headers := []string{"ID", "NAME"}
		rows := make([][]string, 0, len(components))
		for _, c := range components {
			rows = append(rows, []string{c.ComponentID, c.Name})
		}

		if isPicking(cmd) {
			return runPicker(headers, rows, []pickBinding{
				copyColumnBinding("enter", "copy id", "component ID", rows, 0, cmd.OutOrStdout()),
			})
		}

		t := newStyledTable(cmd.OutOrStdout(), headers...)
		for _, r := range rows {
			t.Row(r...)
		}
		return t.Render()
	},
}

var componentsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a component",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		c, err := runComponentsGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), c)
		}

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("componentId", c.ComponentID)
		t.Row("name", c.Name)
		if err := t.Render(); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout())
		return renderLMX(cmd.OutOrStdout(), c.LMX)
	},
}

func init() {
	addPaginationFlags(componentsListCmd)
	addPickFlag(componentsListCmd)
	componentsCmd.AddCommand(componentsListCmd)
	componentsCmd.AddCommand(componentsGetCmd)
	rootCmd.AddCommand(componentsCmd)
}
