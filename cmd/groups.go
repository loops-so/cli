package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

// groupCmdSet binds the four SDK methods that back a *Group resource
// (campaign-groups or transactional-groups) to the labels used in CLI text.
// Both group types share the same Group/CreateGroupRequest/UpdateGroupRequest
// shapes, so the command tree is built once and dispatched twice.
type groupCmdSet struct {
	use         string // top-level command name, e.g. "campaign-groups"
	idLabel     string // table label for the id, e.g. "campaignGroupId"
	singular    string // for help text, e.g. "campaign group"
	runList     func(*config.Config, loops.PaginationParams) ([]loops.Group, error)
	runGet      func(*config.Config, string) (*loops.Group, error)
	runCreate   func(*config.Config, loops.CreateGroupRequest) (*loops.Group, error)
	runUpdate   func(*config.Config, string, loops.UpdateGroupRequest) (*loops.Group, error)
}

// --- SDK method wrappers (one per resource × verb).
// Kept as standalone funcs so tests can call them directly.

func runCampaignGroupsList(cfg *config.Config, params loops.PaginationParams) ([]loops.Group, error) {
	return paginateGroups(params, newAPIClient(cfg).ListCampaignGroups)
}

func runCampaignGroupsGet(cfg *config.Config, id string) (*loops.Group, error) {
	return newAPIClient(cfg).GetCampaignGroup(id)
}

func runCampaignGroupsCreate(cfg *config.Config, req loops.CreateGroupRequest) (*loops.Group, error) {
	return newAPIClient(cfg).CreateCampaignGroup(req)
}

func runCampaignGroupsUpdate(cfg *config.Config, id string, req loops.UpdateGroupRequest) (*loops.Group, error) {
	return newAPIClient(cfg).UpdateCampaignGroup(id, req)
}

func runTransactionalGroupsList(cfg *config.Config, params loops.PaginationParams) ([]loops.Group, error) {
	return paginateGroups(params, newAPIClient(cfg).ListTransactionalGroups)
}

func runTransactionalGroupsGet(cfg *config.Config, id string) (*loops.Group, error) {
	return newAPIClient(cfg).GetTransactionalGroup(id)
}

func runTransactionalGroupsCreate(cfg *config.Config, req loops.CreateGroupRequest) (*loops.Group, error) {
	return newAPIClient(cfg).CreateTransactionalGroup(req)
}

func runTransactionalGroupsUpdate(cfg *config.Config, id string, req loops.UpdateGroupRequest) (*loops.Group, error) {
	return newAPIClient(cfg).UpdateTransactionalGroup(id, req)
}

// paginateGroups runs single-page fetch when a cursor is set, otherwise walks
// every page via loops.Paginate.
func paginateGroups(params loops.PaginationParams, list func(loops.PaginationParams) ([]loops.Group, *loops.Pagination, error)) ([]loops.Group, error) {
	if params.Cursor != "" {
		groups, _, err := list(params)
		return groups, err
	}
	return loops.Paginate(func(cursor string) ([]loops.Group, *loops.Pagination, error) {
		return list(loops.PaginationParams{PerPage: params.PerPage, Cursor: cursor})
	})
}

func printGroup(cmd *cobra.Command, idLabel string, g *loops.Group) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row(idLabel, g.ID)
	t.Row("name", g.Name)
	t.Row("description", g.Description)
	t.Row("createdAt", g.CreatedAt)
	t.Row("updatedAt", g.UpdatedAt)
	return t.Render()
}

func newGroupsCmd(set groupCmdSet) *cobra.Command {
	parent := &cobra.Command{
		Use:   set.use,
		Short: "Manage " + set.use,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List " + set.use,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validatePickFlags(cmd); err != nil {
				return err
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			groups, err := set.runList(cfg, paginationParams(cmd))
			if err != nil {
				return err
			}

			if isJSONOutput() {
				if groups == nil {
					groups = []loops.Group{}
				}
				return printJSON(cmd.OutOrStdout(), groups)
			}

			if len(groups) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No %s found.\n", set.use)
				return nil
			}

			headers := []string{"ID", "NAME", "DESCRIPTION", "UPDATED"}
			rows := make([][]string, 0, len(groups))
			for _, g := range groups {
				rows = append(rows, []string{g.ID, g.Name, g.Description, g.UpdatedAt})
			}

			if isPicking(cmd) {
				return runPicker(headers, rows, []pickBinding{
					copyColumnBinding("enter", "copy id", set.singular+" ID", rows, 0, cmd.OutOrStdout()),
				})
			}

			t := newStyledTable(cmd.OutOrStdout(), headers...)
			for _, r := range rows {
				t.Row(r...)
			}
			return t.Render()
		},
	}
	addPaginationFlags(listCmd)
	addPickFlag(listCmd)

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a " + set.singular,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			g, err := set.runGet(cfg, args[0])
			if err != nil {
				return err
			}
			if isJSONOutput() {
				return printJSON(cmd.OutOrStdout(), g)
			}
			return printGroup(cmd, set.idLabel, g)
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a " + set.singular,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")

			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			g, err := set.runCreate(cfg, loops.CreateGroupRequest{
				Name:        name,
				Description: description,
			})
			if err != nil {
				return err
			}
			if isJSONOutput() {
				return printJSON(cmd.OutOrStdout(), g)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created. (id: %s)\n\n", g.ID)
			return printGroup(cmd, set.idLabel, g)
		},
	}
	createCmd.Flags().StringP("name", "n", "", set.singular+" name (required)")
	createCmd.Flags().StringP("description", "d", "", set.singular+" description")
	createCmd.MarkFlagRequired("name")

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a " + set.singular,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := loops.UpdateGroupRequest{}
			if cmd.Flags().Changed("name") {
				req.Name, _ = cmd.Flags().GetString("name")
			}
			if cmd.Flags().Changed("description") {
				req.Description, _ = cmd.Flags().GetString("description")
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			g, err := set.runUpdate(cfg, args[0], req)
			if err != nil {
				return err
			}
			if isJSONOutput() {
				return printJSON(cmd.OutOrStdout(), g)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated. (id: %s)\n\n", g.ID)
			return printGroup(cmd, set.idLabel, g)
		},
	}
	updateCmd.Flags().StringP("name", "n", "", "New name")
	updateCmd.Flags().StringP("description", "d", "", "New description")
	updateCmd.MarkFlagsOneRequired("name", "description")

	parent.AddCommand(listCmd)
	parent.AddCommand(getCmd)
	parent.AddCommand(createCmd)
	parent.AddCommand(updateCmd)
	return parent
}

func init() {
	rootCmd.AddCommand(newGroupsCmd(groupCmdSet{
		use:       "campaign-groups",
		idLabel:   "campaignGroupId",
		singular:  "campaign group",
		runList:   runCampaignGroupsList,
		runGet:    runCampaignGroupsGet,
		runCreate: runCampaignGroupsCreate,
		runUpdate: runCampaignGroupsUpdate,
	}))

	rootCmd.AddCommand(newGroupsCmd(groupCmdSet{
		use:       "transactional-groups",
		idLabel:   "transactionalGroupId",
		singular:  "transactional group",
		runList:   runTransactionalGroupsList,
		runGet:    runTransactionalGroupsGet,
		runCreate: runTransactionalGroupsCreate,
		runUpdate: runTransactionalGroupsUpdate,
	}))
}
