package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func runContactPropertiesList(cfg *config.Config, customOnly bool) ([]loops.ContactProperty, error) {
	return newAPIClient(cfg).ListContactProperties(customOnly)
}

func runContactPropertiesCreate(cfg *config.Config, name, propType string) error {
	return newAPIClient(cfg).CreateContactProperty(name, propType)
}

var contactPropertiesCmd = &cobra.Command{
	Use:   "contact-properties",
	Short: "Manage contact properties",
}

var contactPropertiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List contact properties",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePickFlags(cmd); err != nil {
			return err
		}

		customOnly, _ := cmd.Flags().GetBool("custom")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		props, err := runContactPropertiesList(cfg, customOnly)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if props == nil {
				props = []loops.ContactProperty{}
			}
			return printJSON(cmd.OutOrStdout(), props)
		}

		if len(props) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No contact properties found.")
			return nil
		}

		headers := []string{"KEY", "LABEL", "TYPE"}
		rows := make([][]string, 0, len(props))
		for _, p := range props {
			rows = append(rows, []string{p.Key, p.Label, p.Type})
		}

		if isPicking(cmd) {
			return runPicker(headers, rows, []pickBinding{
				copyColumnBinding("enter", "copy key", "property key", rows, 0, cmd.OutOrStdout()),
			})
		}

		t := newStyledTable(cmd.OutOrStdout(), headers...)
		for _, r := range rows {
			t.Row(r...)
		}
		return t.Render()
	},
}

var contactPropertiesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a contact property",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		propType, _ := cmd.Flags().GetString("type")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if err := runContactPropertiesCreate(cfg, name, propType); err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), Result{Success: true})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Created.")
		return nil
	},
}

func init() {
	contactPropertiesListCmd.Flags().Bool("custom", false, "Only list custom properties")
	addPickFlag(contactPropertiesListCmd)
	contactPropertiesCmd.AddCommand(contactPropertiesListCmd)

	contactPropertiesCreateCmd.Flags().String("name", "", "Property name (camelCase, e.g. planName)")
	contactPropertiesCreateCmd.Flags().String("type", "", "Property type")
	contactPropertiesCreateCmd.MarkFlagRequired("name")
	contactPropertiesCreateCmd.MarkFlagRequired("type")
	contactPropertiesCmd.AddCommand(contactPropertiesCreateCmd)

	rootCmd.AddCommand(contactPropertiesCmd)
}
