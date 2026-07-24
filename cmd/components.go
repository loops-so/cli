package cmd

import (
	"fmt"
	"os"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func runComponentsGet(cfg *config.Config, id string) (*loops.Component, error) {
	return newAPIClient(cfg).GetComponent(id)
}

func runComponentsCreate(cfg *config.Config, req loops.CreateComponentRequest) (*loops.Component, error) {
	return newAPIClient(cfg).CreateComponent(req)
}

func runComponentsUpdate(cfg *config.Config, id string, req loops.UpdateComponentRequest) (*loops.UpdateComponentResult, error) {
	return newAPIClient(cfg).UpdateComponent(id, req)
}

// lmxFromCmd resolves the LMX body from either --lmx (inline) or --lmx-file
// (raw file contents). The two flags are mutually exclusive. The second return
// value reports whether either flag was set.
func lmxFromCmd(cmd *cobra.Command) (string, bool, error) {
	if cmd.Flags().Changed("lmx") {
		lmx, _ := cmd.Flags().GetString("lmx")
		return lmx, true, nil
	}
	if cmd.Flags().Changed("lmx-file") {
		path, _ := cmd.Flags().GetString("lmx-file")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", false, fmt.Errorf("read --lmx-file: %w", err)
		}
		return string(data), true, nil
	}
	return "", false, nil
}

// printComponent renders a component as a field table followed by its
// highlighted LMX body, matching the `components get` output.
func printComponent(cmd *cobra.Command, c *loops.Component) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("componentId", c.ID)
	t.Row("name", c.Name)
	if err := t.Render(); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout())
	return renderLMX(cmd.OutOrStdout(), c.LMX)
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
			rows = append(rows, []string{c.ID, c.Name})
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

		return printComponent(cmd, c)
	},
}

var componentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a component",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		lmx, _, err := lmxFromCmd(cmd)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		c, err := runComponentsCreate(cfg, loops.CreateComponentRequest{
			Name: name,
			LMX:  lmx,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), c)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created. (componentId: %s)\n\n", c.ID)
		return printComponent(cmd, c)
	},
}

var componentsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a component",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		lmx, lmxSet, err := lmxFromCmd(cmd)
		if err != nil {
			return err
		}

		req := loops.UpdateComponentRequest{}
		if cmd.Flags().Changed("name") {
			req.Name, _ = cmd.Flags().GetString("name")
		}
		if lmxSet {
			req.LMX = lmx
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		result, err := runComponentsUpdate(cfg, args[0], req)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), result)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (componentId: %s, affectedEmailCount: %d)\n\n", result.ID, result.AffectedEmailCount)
		return printComponent(cmd, &result.Component)
	},
}

func init() {
	addPaginationFlags(componentsListCmd)
	addPickFlag(componentsListCmd)
	componentsCmd.AddCommand(componentsListCmd)
	componentsCmd.AddCommand(componentsGetCmd)

	componentsCreateCmd.Flags().String("name", "", "Component name")
	componentsCreateCmd.Flags().String("lmx", "", "LMX markup (inline)")
	componentsCreateCmd.Flags().String("lmx-file", "", "Path to a file containing LMX markup")
	componentsCreateCmd.MarkFlagRequired("name")
	componentsCreateCmd.MarkFlagsMutuallyExclusive("lmx", "lmx-file")
	componentsCreateCmd.MarkFlagsOneRequired("lmx", "lmx-file")
	componentsCmd.AddCommand(componentsCreateCmd)

	componentsUpdateCmd.Flags().String("name", "", "Component name")
	componentsUpdateCmd.Flags().String("lmx", "", "LMX markup (inline)")
	componentsUpdateCmd.Flags().String("lmx-file", "", "Path to a file containing LMX markup")
	componentsUpdateCmd.MarkFlagsMutuallyExclusive("lmx", "lmx-file")
	componentsUpdateCmd.MarkFlagsOneRequired("name", "lmx", "lmx-file")
	componentsCmd.AddCommand(componentsUpdateCmd)

	rootCmd.AddCommand(componentsCmd)
}
