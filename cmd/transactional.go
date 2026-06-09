package cmd

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/loops-so/cli/internal/cmdutil"
	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func parseDataVars(vars []string, jsonFile string) (map[string]any, error) {
	var m map[string]any
	if jsonFile != "" {
		var err error
		m, err = cmdutil.ParseJSONFile("json-vars", jsonFile)
		if err != nil {
			return nil, err
		}
	}
	return cmdutil.ParseKeyValuePairs("var", vars, m)
}

func attachmentFromPath(path string) (loops.Attachment, error) {
	info, err := os.Stat(path)
	if err != nil {
		return loops.Attachment{}, fmt.Errorf("attachment %q: %w", path, err)
	}
	if info.IsDir() {
		return loops.Attachment{}, fmt.Errorf("attachment %q: is a directory", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return loops.Attachment{}, fmt.Errorf("attachment %q: %w", path, err)
	}

	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	contentType := http.DetectContentType(sniff)

	return loops.Attachment{
		Filename:    filepath.Base(path),
		ContentType: contentType,
		Data:        base64.StdEncoding.EncodeToString(data),
	}, nil
}

func runTransactionalList(cfg *config.Config, params loops.PaginationParams) ([]loops.Transactional, error) {
	client := newAPIClient(cfg)
	if params.Cursor != "" {
		emails, _, err := client.ListTransactionals(params)
		return emails, err
	}
	return loops.Paginate(func(cursor string) ([]loops.Transactional, *loops.Pagination, error) {
		return client.ListTransactionals(loops.PaginationParams{
			PerPage: params.PerPage,
			Cursor:  cursor,
		})
	})
}

func runTransactionalGet(cfg *config.Config, id string) (*loops.Transactional, error) {
	return newAPIClient(cfg).GetTransactional(id)
}

func runTransactionalCreate(cfg *config.Config, req loops.CreateTransactionalRequest) (*loops.TransactionalDraft, error) {
	return newAPIClient(cfg).CreateTransactional(req)
}

func runTransactionalUpdate(cfg *config.Config, id string, req loops.UpdateTransactionalRequest) (*loops.Transactional, error) {
	return newAPIClient(cfg).UpdateTransactional(id, req)
}

func runTransactionalDraft(cfg *config.Config, id string) (*loops.TransactionalDraft, error) {
	return newAPIClient(cfg).EnsureTransactionalDraft(id)
}

func runTransactionalPublish(cfg *config.Config, id string) (*loops.Transactional, error) {
	return newAPIClient(cfg).PublishTransactional(id)
}

func runTransactionalSend(cfg *config.Config, req loops.SendTransactionalRequest) error {
	return newAPIClient(cfg).SendTransactional(req)
}

var transactionalCmd = &cobra.Command{
	Use:   "transactional",
	Short: "Manage transactional emails",
}

var transactionalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List transactional emails",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePickFlags(cmd); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		emails, err := runTransactionalList(cfg, paginationParams(cmd))
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if emails == nil {
				emails = []loops.Transactional{}
			}
			return printJSON(cmd.OutOrStdout(), emails)
		}

		if len(emails) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No transactional emails found.")
			return nil
		}

		headers := []string{"ID", "NAME", "DRAFT MSG", "PUBLISHED MSG", "UPDATED"}
		rows := make([][]string, 0, len(emails))
		for _, e := range emails {
			rows = append(rows, []string{
				e.ID,
				e.Name,
				deref(e.DraftEmailMessageID),
				deref(e.PublishedEmailMessageID),
				e.UpdatedAt,
			})
		}

		if isPicking(cmd) {
			out := cmd.OutOrStdout()
			return runPicker(headers, rows, []pickBinding{
				copyColumnBinding("enter", "copy id", "transactional ID", rows, 0, out),
				copyColumnBinding("alt-enter", "copy publishedEmailMessageId", "published message ID", rows, 3, out),
			})
		}

		t := newStyledTable(cmd.OutOrStdout(), headers...)
		for _, r := range rows {
			t.Row(r...)
		}
		return t.Render()
	},
}

var transactionalGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a transactional email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		tx, err := runTransactionalGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), tx)
		}

		return printTransactional(cmd, tx)
	},
}

var transactionalCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a transactional email with an empty draft",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		tx, err := runTransactionalCreate(cfg, loops.CreateTransactionalRequest{Name: name})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), tx)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created. (id: %s, draftEmailMessageId: %s, contentRevisionId: %s)\n\n",
			tx.ID, deref(tx.DraftEmailMessageID), deref(tx.DraftEmailMessageContentRevisionID))
		return printTransactional(cmd, &tx.Transactional)
	},
}

var transactionalUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a transactional email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		tx, err := runTransactionalUpdate(cfg, args[0], loops.UpdateTransactionalRequest{Name: name})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), tx)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (id: %s)\n\n", tx.ID)
		return printTransactional(cmd, tx)
	},
}

var transactionalDraftCmd = &cobra.Command{
	Use:   "draft <id>",
	Short: "Ensure a draft exists for a transactional email",
	Long: "Ensures the transactional email has a draft. If a draft already exists it is returned unchanged;\n" +
		"otherwise a new empty draft is created (seeded from the most recent published version when present).",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		tx, err := runTransactionalDraft(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), tx)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Draft ready. (id: %s, draftEmailMessageId: %s, contentRevisionId: %s)\n\n",
			tx.ID, deref(tx.DraftEmailMessageID), deref(tx.DraftEmailMessageContentRevisionID))
		return printTransactional(cmd, &tx.Transactional)
	},
}

var transactionalPublishCmd = &cobra.Command{
	Use:   "publish <id>",
	Short: "Publish the current draft of a transactional email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		tx, err := runTransactionalPublish(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), tx)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Published. (id: %s, publishedEmailMessageId: %s)\n\n",
			tx.ID, deref(tx.PublishedEmailMessageID))
		return printTransactional(cmd, tx)
	},
}

func printTransactional(cmd *cobra.Command, tx *loops.Transactional) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("transactionalId", tx.ID)
	t.Row("name", tx.Name)
	t.Row("draftEmailMessageId", deref(tx.DraftEmailMessageID))
	t.Row("publishedEmailMessageId", deref(tx.PublishedEmailMessageID))
	t.Row("dataVariables", strings.Join(tx.DataVariables, ", "))
	t.Row("createdAt", tx.CreatedAt)
	t.Row("updatedAt", tx.UpdatedAt)
	return t.Render()
}

func transactionalSendRunE(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	id := args[0]
	email, _ := cmd.Flags().GetString("email")
	idempotencyKey, _ := cmd.Flags().GetString("idempotency-key")

	req := loops.SendTransactionalRequest{
		Email:           email,
		TransactionalID: id,
		IdempotencyKey:  idempotencyKey,
	}

	if cmd.Flags().Changed("add-to-audience") {
		v, _ := cmd.Flags().GetBool("add-to-audience")
		req.AddToAudience = &v
	}

	varPairs, _ := cmd.Flags().GetStringArray("var")
	jsonFile, _ := cmd.Flags().GetString("json-vars")
	dataVars, err := parseDataVars(varPairs, jsonFile)
	if err != nil {
		return err
	}
	if len(dataVars) > 0 {
		req.DataVariables = dataVars
	}

	paths, _ := cmd.Flags().GetStringArray("attachment")
	for _, path := range paths {
		a, err := attachmentFromPath(path)
		if err != nil {
			return err
		}
		req.Attachments = append(req.Attachments, a)
	}

	if err := runTransactionalSend(cfg, req); err != nil {
		return err
	}

	if isJSONOutput() {
		return printJSON(cmd.OutOrStdout(), Result{Success: true})
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Sent.")
	return nil
}

func addTransactionalSendFlags(cmd *cobra.Command) {
	cmd.Flags().String("email", "", "Recipient email address")
	cmd.Flags().BoolP("add-to-audience", "a", false, "Create a contact if one doesn't exist")
	cmd.Flags().StringArrayP("var", "v", nil, "Data variable as KEY=value (repeatable)")
	cmd.Flags().StringP("json-vars", "j", "", "Path to a JSON file of data variables")
	cmd.Flags().StringArrayP("attachment", "A", nil, "Path to a file to attach (repeatable)")
	cmd.Flags().String("idempotency-key", "", "Idempotency key to prevent duplicate sends")
	cmd.MarkFlagRequired("email")
}

var transactionalSendCmd = &cobra.Command{
	Use:   "send <id>",
	Short: "Send a transactional email",
	Args:  cobra.ExactArgs(1),
	RunE:  transactionalSendRunE,
}

func init() {
	addPaginationFlags(transactionalListCmd)
	addPickFlag(transactionalListCmd)
	transactionalCmd.AddCommand(transactionalListCmd)

	transactionalCmd.AddCommand(transactionalGetCmd)

	transactionalCreateCmd.Flags().StringP("name", "n", "", "Transactional email name (required)")
	transactionalCreateCmd.MarkFlagRequired("name")
	transactionalCmd.AddCommand(transactionalCreateCmd)

	transactionalUpdateCmd.Flags().StringP("name", "n", "", "Transactional email name (required)")
	transactionalUpdateCmd.MarkFlagRequired("name")
	transactionalCmd.AddCommand(transactionalUpdateCmd)

	transactionalCmd.AddCommand(transactionalDraftCmd)
	transactionalCmd.AddCommand(transactionalPublishCmd)

	addTransactionalSendFlags(transactionalSendCmd)
	transactionalCmd.AddCommand(transactionalSendCmd)

	rootCmd.AddCommand(transactionalCmd)
}
