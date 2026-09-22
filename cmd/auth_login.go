package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginSkipVerify bool

var loginCmd = &cobra.Command{
	Use:   "login <name>",
	Short: "Authenticate with your Loops API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		fmt.Fprintln(os.Stderr, "Get your API key at https://app.loops.so/settings?page=api")
		fmt.Fprint(os.Stderr, "Enter your API key: ")
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}

		apiKey := strings.TrimSpace(string(raw))
		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}

		result, err := runAuthLogin(apiKey, name, loginSkipVerify)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			msg := fmt.Sprintf("API key saved as %q", name)
			if result != nil {
				msg = fmt.Sprintf("Authenticated as team: %s", result.TeamName)
			}
			return printJSON(cmd.OutOrStdout(), Result{Success: true, Message: msg})
		}
		if result != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "API key saved as %q. Authenticated as team: %s\n", name, result.TeamName)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "API key saved as %q.\n", name)
		}
		return nil
	},
}

func runAuthLogin(apiKey, name string, skipVerify bool) (*loops.APIKeyResponse, error) {
	if name == "" {
		return nil, errors.New("a key name is required")
	}
	if skipVerify {
		if err := config.Save(apiKey, name); err != nil {
			return nil, err
		}
		return nil, nil
	}
	result, err := newAPIClient(&config.Config{EndpointURL: config.EndpointURL(), APIKey: apiKey, Debug: debugFlag}).GetAPIKey()
	if err != nil {
		return nil, fmt.Errorf("API key verification failed: %w", err)
	}
	if err := config.Save(apiKey, name); err != nil {
		return nil, err
	}
	return result, nil
}

func init() {
	loginCmd.Flags().BoolVar(&loginSkipVerify, "skip-verify", false, "Save the API key without verifying it")
	authCmd.AddCommand(loginCmd)
}
