package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/loops-so/cli/internal/cmdutil"
	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

// params common to create and update
type contactFieldParams struct {
	FirstName         string
	LastName          string
	Subscribed        *bool
	UserGroup         string
	MailingLists      map[string]bool
	ContactProperties map[string]any
}

// flags common to create and update
func addContactFieldFlags(cmd *cobra.Command) {
	cmd.Flags().String("first-name", "", "First name")
	cmd.Flags().String("last-name", "", "Last name")
	cmd.Flags().StringP("subscribed", "s", "", `Subscribed status ("true" or "false")`)
	cmd.Flags().String("user-group", "", "User group")
	cmd.Flags().StringArray("list", nil, "Mailing list subscription as id=true|false (repeatable)")
	cmd.Flags().StringArray("prop", nil, "Contact property as KEY=value (repeatable)")
	cmd.Flags().String("contact-props", "", "Path to a JSON file of contact properties")
}

// read common flags
func contactFieldParamsFromCmd(cmd *cobra.Command) (contactFieldParams, error) {
	firstName, _ := cmd.Flags().GetString("first-name")
	lastName, _ := cmd.Flags().GetString("last-name")
	userGroup, _ := cmd.Flags().GetString("user-group")

	params := contactFieldParams{
		FirstName: firstName,
		LastName:  lastName,
		UserGroup: userGroup,
	}

	if cmd.Flags().Changed("subscribed") {
		subStr, _ := cmd.Flags().GetString("subscribed")
		switch subStr {
		case "true":
			b := true
			params.Subscribed = &b
		case "false":
			b := false
			params.Subscribed = &b
		default:
			return params, fmt.Errorf("--subscribed must be \"true\" or \"false\"")
		}
	}

	listPairs, _ := cmd.Flags().GetStringArray("list")
	mailingLists, err := parseMailingLists(listPairs)
	if err != nil {
		return params, err
	}
	params.MailingLists = mailingLists

	if contactPropsPath, _ := cmd.Flags().GetString("contact-props"); contactPropsPath != "" {
		contactProps, err := cmdutil.ParseJSONFile("contact-props", contactPropsPath)
		if err != nil {
			return params, err
		}
		params.ContactProperties = contactProps
	}
	propPairs, _ := cmd.Flags().GetStringArray("prop")
	props, err := cmdutil.ParseKeyValuePairs("prop", propPairs, params.ContactProperties)
	if err != nil {
		return params, err
	}
	params.ContactProperties = props

	return params, nil
}

// find

func runContactsFind(cfg *config.Config, email, userID string) ([]loops.Contact, error) {
	return newAPIClient(cfg).FindContacts(loops.FindContactParams{
		Email:  email,
		UserID: userID,
	})
}

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage contacts",
}

var contactsFindCmd = &cobra.Command{
	Use:   "find",
	Short: "Find a contact by email or user ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		userID, _ := cmd.Flags().GetString("user-id")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		contacts, err := runContactsFind(cfg, email, userID)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if contacts == nil {
				contacts = []loops.Contact{}
			}
			return printJSON(cmd.OutOrStdout(), contacts)
		}

		if len(contacts) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No contacts found.")
			return nil
		}

		c := contacts[0]
		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE", "CUSTOM")
		row := func(field, value string, custom bool) {
			marker := ""
			if custom {
				marker = "*"
			}
			t.Row(field, value, marker)
		}
		row("id", c.ID, false)
		row("email", c.Email, false)
		row("firstName", deref(c.FirstName), false)
		row("lastName", deref(c.LastName), false)
		row("subscribed", strconv.FormatBool(c.Subscribed), false)
		row("source", c.Source, false)
		row("userGroup", c.UserGroup, false)
		row("userId", deref(c.UserID), false)
		row("optInStatus", deref(c.OptInStatus), false)
		row("mailingLists", formatMailingLists(c.MailingLists), false)
		for _, k := range sortedKeys(c.Custom) {
			row(k, formatCustomValue(c.Custom[k]), true)
		}
		return t.Render()
	},
}

// create

type contactCreateResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

func runContactsCreate(cfg *config.Config, req loops.CreateContactRequest) (string, error) {
	return newAPIClient(cfg).CreateContact(req)
}

var contactsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a contact",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		email, _ := cmd.Flags().GetString("email")
		source, _ := cmd.Flags().GetString("source")
		userID, _ := cmd.Flags().GetString("user-id")

		fields, err := contactFieldParamsFromCmd(cmd)
		if err != nil {
			return err
		}

		id, err := runContactsCreate(cfg, loops.CreateContactRequest{
			Email:             email,
			FirstName:         fields.FirstName,
			LastName:          fields.LastName,
			Source:            source,
			Subscribed:        fields.Subscribed,
			UserGroup:         fields.UserGroup,
			UserID:            userID,
			MailingLists:      fields.MailingLists,
			ContactProperties: fields.ContactProperties,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), contactCreateResult{Success: true, ID: id})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created. (id: %s)\n", id)
		return nil
	},
}

// update

func runContactsUpdate(cfg *config.Config, req loops.UpdateContactRequest) error {
	return newAPIClient(cfg).UpdateContact(req)
}

var contactsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a contact",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		userID, _ := cmd.Flags().GetString("user-id")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		fields, err := contactFieldParamsFromCmd(cmd)
		if err != nil {
			return err
		}

		if err := runContactsUpdate(cfg, loops.UpdateContactRequest{
			Email:             email,
			UserID:            userID,
			FirstName:         fields.FirstName,
			LastName:          fields.LastName,
			Subscribed:        fields.Subscribed,
			UserGroup:         fields.UserGroup,
			MailingLists:      fields.MailingLists,
			ContactProperties: fields.ContactProperties,
		}); err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), Result{Success: true})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Updated.")
		return nil
	},
}

// delete

func runContactsDelete(cfg *config.Config, email, userID string) error {
	return newAPIClient(cfg).DeleteContact(email, userID)
}

var contactsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a contact",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		userID, _ := cmd.Flags().GetString("user-id")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if err := runContactsDelete(cfg, email, userID); err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), Result{Success: true})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Deleted.")
		return nil
	},
}

func formatMailingLists(m map[string]bool) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatCustomValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func init() {
	contactsFindCmd.Flags().StringP("email", "e", "", "Contact email address")
	contactsFindCmd.Flags().StringP("user-id", "u", "", "Contact user ID")
	contactsFindCmd.MarkFlagsOneRequired("email", "user-id")
	contactsFindCmd.MarkFlagsMutuallyExclusive("email", "user-id")
	contactsCmd.AddCommand(contactsFindCmd)

	contactsCreateCmd.Flags().StringP("email", "e", "", "Contact email address")
	contactsCreateCmd.Flags().String("source", "", "Source")
	contactsCreateCmd.Flags().StringP("user-id", "u", "", "User ID")
	addContactFieldFlags(contactsCreateCmd)
	contactsCreateCmd.MarkFlagRequired("email")
	contactsCmd.AddCommand(contactsCreateCmd)

	contactsUpdateCmd.Flags().StringP("email", "e", "", "Contact email address")
	contactsUpdateCmd.Flags().StringP("user-id", "u", "", "User ID")
	addContactFieldFlags(contactsUpdateCmd)
	contactsUpdateCmd.MarkFlagsOneRequired("email", "user-id")
	contactsCmd.AddCommand(contactsUpdateCmd)

	contactsDeleteCmd.Flags().StringP("email", "e", "", "Contact email address")
	contactsDeleteCmd.Flags().StringP("user-id", "u", "", "Contact user ID")
	contactsDeleteCmd.MarkFlagsOneRequired("email", "user-id")
	contactsDeleteCmd.MarkFlagsMutuallyExclusive("email", "user-id")
	contactsCmd.AddCommand(contactsDeleteCmd)

	rootCmd.AddCommand(contactsCmd)
}
