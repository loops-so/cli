package cmd

import (
	"fmt"
	"strings"

	"github.com/loops-so/cli/internal/cmdutil"
	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func parseMailingLists(pairs []string) (map[string]bool, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	m := make(map[string]bool, len(pairs))
	for _, pair := range pairs {
		idx := strings.IndexByte(pair, '=')
		if idx < 0 {
			return nil, fmt.Errorf("--list %q: expected id=true|false", pair)
		}
		id := pair[:idx]
		val := strings.ToLower(pair[idx+1:])
		switch val {
		case "true":
			m[id] = true
		case "false":
			m[id] = false
		default:
			return nil, fmt.Errorf("--list %q: value must be \"true\" or \"false\"", pair)
		}
	}
	return m, nil
}

func runEventsSend(cfg *config.Config, req loops.SendEventRequest) error {
	return newAPIClient(cfg).SendEvent(req)
}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Manage events",
}

var eventsSendCmd = &cobra.Command{
	Use:   "send <event>",
	Short: "Send an event",
	Args:  cobra.ExactArgs(1),
	RunE:  eventsSendRunE,
}

func eventsSendRunE(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	email, _ := cmd.Flags().GetString("email")
	userID, _ := cmd.Flags().GetString("user-id")

	eventName := args[0]
	idempotencyKey, _ := cmd.Flags().GetString("idempotency-key")

	req := loops.SendEventRequest{
		Email:          email,
		UserID:         userID,
		EventName:      eventName,
		IdempotencyKey: idempotencyKey,
	}

	if propsPath, _ := cmd.Flags().GetString("props"); propsPath != "" {
		props, err := cmdutil.ParseJSONFile("props", propsPath)
		if err != nil {
			return err
		}
		if nested, ok := props["eventProperties"]; ok {
			if m, ok := nested.(map[string]any); ok {
				req.EventProperties = m
			}
		} else {
			req.EventProperties = props
		}
	}

	if contactPropsPath, _ := cmd.Flags().GetString("contact-props"); contactPropsPath != "" {
		contactProps, err := cmdutil.ParseJSONFile("contact-props", contactPropsPath)
		if err != nil {
			return err
		}
		req.ContactProperties = contactProps
	}

	listPairs, _ := cmd.Flags().GetStringArray("list")
	mailingLists, err := parseMailingLists(listPairs)
	if err != nil {
		return err
	}
	req.MailingLists = mailingLists

	if err := runEventsSend(cfg, req); err != nil {
		return err
	}

	if isJSONOutput() {
		return printJSON(cmd.OutOrStdout(), Result{Success: true})
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Sent.")
	return nil
}

func addEventsSendFlags(cmd *cobra.Command) {
	cmd.Flags().String("email", "", "Contact email address")
	cmd.Flags().String("user-id", "", "Contact user ID")
	cmd.Flags().String("props", "", "Path to a JSON file of event properties")
	cmd.Flags().String("contact-props", "", "Path to a JSON file of contact properties")
	cmd.Flags().StringArray("list", nil, "Mailing list subscription as id=true|false (repeatable)")
	cmd.Flags().String("idempotency-key", "", "Idempotency key to prevent duplicate sends")
}

func init() {
	addEventsSendFlags(eventsSendCmd)
	eventsSendCmd.MarkFlagsOneRequired("email", "user-id")
	eventsCmd.AddCommand(eventsSendCmd)
	rootCmd.AddCommand(eventsCmd)
}
