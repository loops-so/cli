package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/colorprofile"
	"github.com/loops-so/loops-go"
	"github.com/loops-so/cli/internal/config"
	"github.com/spf13/cobra"
)

func fromEmailUsername(s string) string {
	before, _, _ := strings.Cut(s, "@")
	return before
}

// emailMessageFieldParams holds the six content fields shared by
// `campaigns create` and `email-messages update`. Set records which fields the
// user explicitly provided (keyed by JSON field name) so partial updates can
// send only those fields.
type emailMessageFieldParams struct {
	Subject      string
	PreviewText  string
	FromName     string
	FromEmail    string
	ReplyToEmail string
	LMX          string
	Set          map[string]bool
}

func addEmailMessageFieldFlags(cmd *cobra.Command) {
	cmd.Flags().String("subject", "", "Email subject")
	cmd.Flags().String("preview-text", "", "Email preview text")
	cmd.Flags().String("from-name", "", "Sender name")
	cmd.Flags().String("from-email", "", "Username only: a@example.com -> a")
	cmd.Flags().String("reply-to", "", "Reply-to email address")
	cmd.Flags().String("lmx", "", "LMX markup (inline)")
	cmd.Flags().String("lmx-file", "", "Path to a file containing LMX markup")
	cmd.MarkFlagsMutuallyExclusive("lmx", "lmx-file")
}

func emailMessageFieldParamsFromCmd(cmd *cobra.Command) (emailMessageFieldParams, error) {
	p := emailMessageFieldParams{Set: map[string]bool{}}

	if cmd.Flags().Changed("subject") {
		p.Subject, _ = cmd.Flags().GetString("subject")
		p.Set["subject"] = true
	}
	if cmd.Flags().Changed("preview-text") {
		p.PreviewText, _ = cmd.Flags().GetString("preview-text")
		p.Set["previewText"] = true
	}
	if cmd.Flags().Changed("from-name") {
		p.FromName, _ = cmd.Flags().GetString("from-name")
		p.Set["fromName"] = true
	}
	if cmd.Flags().Changed("from-email") {
		v, _ := cmd.Flags().GetString("from-email")
		p.FromEmail = fromEmailUsername(v)
		p.Set["fromEmail"] = true
	}
	if cmd.Flags().Changed("reply-to") {
		p.ReplyToEmail, _ = cmd.Flags().GetString("reply-to")
		p.Set["replyToEmail"] = true
	}
	if cmd.Flags().Changed("lmx") {
		p.LMX, _ = cmd.Flags().GetString("lmx")
		p.Set["lmx"] = true
	}
	if cmd.Flags().Changed("lmx-file") {
		path, _ := cmd.Flags().GetString("lmx-file")
		data, err := os.ReadFile(path)
		if err != nil {
			return p, fmt.Errorf("read --lmx-file: %w", err)
		}
		p.LMX = string(data)
		p.Set["lmx"] = true
	}
	return p, nil
}

func runEmailMessagesGet(cfg *config.Config, id string) (*loops.EmailMessage, error) {
	return newAPIClient(cfg).GetEmailMessage(id)
}

func runEmailMessagesUpdate(cfg *config.Config, id string, req loops.UpdateEmailMessageRequest) (*loops.EmailMessage, error) {
	return newAPIClient(cfg).UpdateEmailMessage(id, req)
}

func fetchLatestRevisionID(cfg *config.Config, id string) (string, error) {
	msg, err := newAPIClient(cfg).GetEmailMessage(id)
	if err != nil {
		return "", fmt.Errorf("fetch current revision: %w", err)
	}
	return deref(msg.ContentRevisionID), nil
}

var emailMessagesCmd = &cobra.Command{
	Use:    "email-messages",
	Short:  "Manage email messages",
	Hidden: true,
}

var emailMessagesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get an email message",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		msg, err := runEmailMessagesGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), msg)
		}

		return printEmailMessage(cmd, msg)
	},
}

var emailMessagesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a draft email message",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		params, err := emailMessageFieldParamsFromCmd(cmd)
		if err != nil {
			return err
		}

		revisionID, _ := cmd.Flags().GetString("expected-revision-id")
		force, _ := cmd.Flags().GetBool("force")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if force {
			revisionID, err = fetchLatestRevisionID(cfg, args[0])
			if err != nil {
				return err
			}
		}

		req := loops.UpdateEmailMessageRequest{
			EmailMessageFields: loops.EmailMessageFields{
				Subject:      params.Subject,
				PreviewText:  params.PreviewText,
				FromName:     params.FromName,
				FromEmail:    params.FromEmail,
				ReplyToEmail: params.ReplyToEmail,
				LMX:          params.LMX,
			},
			Set:                params.Set,
			ExpectedRevisionID: revisionID,
		}

		msg, err := runEmailMessagesUpdate(cfg, args[0], req)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), msg)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (emailMessageId: %s, contentRevisionId: %s)\n", msg.EmailMessageID, deref(msg.ContentRevisionID))
		fmt.Fprintln(cmd.OutOrStdout())
		if err := printEmailMessage(cmd, msg); err != nil {
			return err
		}
		printLmxWarnings(cmd, msg.Warnings)
		return nil
	},
}

func printEmailMessage(cmd *cobra.Command, msg *loops.EmailMessage) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("emailMessageId", msg.EmailMessageID)
	t.Row("campaignId", deref(msg.CampaignID))
	t.Row("subject", msg.Subject)
	t.Row("previewText", msg.PreviewText)
	t.Row("fromName", msg.FromName)
	t.Row("fromEmail", msg.FromEmail)
	t.Row("replyToEmail", msg.ReplyToEmail)
	t.Row("contentRevisionId", deref(msg.ContentRevisionID))
	t.Row("updatedAt", msg.UpdatedAt)
	if err := t.Render(); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout())
	return renderLMX(cmd.OutOrStdout(), msg.LMX)
}

// renderLMX prints the LMX body with chroma syntax highlighting via the xml
// lexer (LMX is JSX/XML-tag-shaped). The chroma style maps two token kinds to
// fang.ColorScheme colors so highlighting reuses the same palette as the rest
// of the CLI's output.
func renderLMX(out io.Writer, lmx string) error {
	lexer := lexers.Get("xml")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	iterator, err := lexer.Tokenise(nil, lmx)
	if err != nil {
		return err
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, lmxChromaStyle(), iterator); err != nil {
		return err
	}
	cw := colorprofile.NewWriter(out, os.Environ())
	_, err = fmt.Fprintln(cw, strings.TrimRight(buf.String(), "\n"))
	return err
}

func lmxChromaStyle() *chroma.Style {
	cs := fangColorScheme()
	return chroma.MustNewStyle("lmx", chroma.StyleEntries{
		chroma.NameTag:       hexColor(cs.Program),
		chroma.LiteralString: hexColor(cs.QuotedString),
	})
}

func printLmxWarnings(cmd *cobra.Command, warnings []loops.LmxWarning) {
	if len(warnings) == 0 {
		return
	}
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Warnings:")
	for _, warn := range warnings {
		if warn.Path != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %s (%s)\n", warn.Rule, warn.Message, warn.Path)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %s\n", warn.Rule, warn.Message)
		}
	}
}

func init() {
	emailMessagesCmd.AddCommand(emailMessagesGetCmd)

	addEmailMessageFieldFlags(emailMessagesUpdateCmd)
	emailMessagesUpdateCmd.Flags().StringP("expected-revision-id", "r", "", "Last-seen contentRevisionId. Get this from a prior 'email-messages get'. Mutually exclusive with --force.")
	emailMessagesUpdateCmd.Flags().BoolP("force", "f", false, "Fetch the current revision and use it (overwrites any concurrent edits). Mutually exclusive with --expected-revision-id.")
	emailMessagesUpdateCmd.MarkFlagsMutuallyExclusive("expected-revision-id", "force")
	emailMessagesUpdateCmd.MarkFlagsOneRequired("expected-revision-id", "force")
	emailMessagesUpdateCmd.MarkFlagsOneRequired("subject", "preview-text", "from-name", "from-email", "reply-to", "lmx", "lmx-file")
	emailMessagesCmd.AddCommand(emailMessagesUpdateCmd)

	rootCmd.AddCommand(emailMessagesCmd)
}
