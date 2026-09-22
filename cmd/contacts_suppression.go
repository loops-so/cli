package cmd

import (
	"fmt"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

var contactsSuppressionCmd = &cobra.Command{
	Use:   "suppression",
	Short: "Manage contact suppression",
}

// check

func runContactsSuppressionCheck(cfg *config.Config, email, userID string) (*loops.ContactSuppression, error) {
	return newAPIClient(cfg).CheckContactSuppression(email, userID)
}

var contactsSuppressionCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check suppression status for a contact",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		userID, _ := cmd.Flags().GetString("user-id")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		result, err := runContactsSuppressionCheck(cfg, email, userID)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), result)
		}

		suppressed := "no"
		if result.IsSuppressed {
			suppressed = "yes"
		}
		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("isSuppressed", suppressed)
		t.Row("removalQuota", fmt.Sprintf("%d/%d remaining", result.RemovalQuota.Remaining, result.RemovalQuota.Limit))
		return t.Render()
	},
}

// remove

func runContactsSuppressionRemove(cfg *config.Config, email, userID string) (*loops.ContactSuppressionRemoval, error) {
	return newAPIClient(cfg).RemoveContactSuppression(email, userID)
}

var contactsSuppressionRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a contact from the suppression list",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		userID, _ := cmd.Flags().GetString("user-id")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		result, err := runContactsSuppressionRemove(cfg, email, userID)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), result)
		}

		fmt.Fprintln(cmd.OutOrStdout(), result.Message)
		fmt.Fprintf(cmd.OutOrStdout(), "Removal quota: %d/%d remaining\n", result.RemovalQuota.Remaining, result.RemovalQuota.Limit)
		return nil
	},
}

func init() {
	contactsSuppressionCheckCmd.Flags().StringP("email", "e", "", "Contact email address")
	contactsSuppressionCheckCmd.Flags().StringP("user-id", "u", "", "Contact user ID")
	contactsSuppressionCheckCmd.MarkFlagsOneRequired("email", "user-id")
	contactsSuppressionCheckCmd.MarkFlagsMutuallyExclusive("email", "user-id")
	contactsSuppressionCmd.AddCommand(contactsSuppressionCheckCmd)

	contactsSuppressionRemoveCmd.Flags().StringP("email", "e", "", "Contact email address")
	contactsSuppressionRemoveCmd.Flags().StringP("user-id", "u", "", "Contact user ID")
	contactsSuppressionRemoveCmd.MarkFlagsOneRequired("email", "user-id")
	contactsSuppressionRemoveCmd.MarkFlagsMutuallyExclusive("email", "user-id")
	contactsSuppressionCmd.AddCommand(contactsSuppressionRemoveCmd)

	contactsCmd.AddCommand(contactsSuppressionCmd)
}
