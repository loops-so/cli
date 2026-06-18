package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/colorprofile"
	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func fromEmailUsername(s string) string {
	before, _, _ := strings.Cut(s, "@")
	return before
}

// emailMessageFieldParams holds the editable fields shared by
// `campaigns create` and `email-messages update`. Set records which fields the
// user explicitly provided (keyed by JSON field name) so partial updates can
// send only those fields.
//
// The *Fallbacks maps replace the server-side maps wholesale when the
// corresponding Set key is true — they do not merge.
type emailMessageFieldParams struct {
	Subject                    string
	PreviewText                string
	FromName                   string
	FromEmail                  string
	ReplyToEmail               string
	CCEmail                    string
	BCCEmail                   string
	LanguageCode               string
	EmailFormat                string
	LMX                        string
	ContactPropertiesFallbacks map[string]*string
	EventPropertiesFallbacks   map[string]*string
	DataVariablesFallbacks     map[string]*string
	Set                        map[string]bool
}

func addEmailMessageFieldFlags(cmd *cobra.Command) {
	cmd.Flags().String("subject", "", "Email subject")
	cmd.Flags().String("preview-text", "", "Email preview text")
	cmd.Flags().String("from-name", "", "Sender name")
	cmd.Flags().String("from-email", "", "Username only: a@example.com -> a")
	cmd.Flags().String("reply-to", "", "Reply-to email address")
	cmd.Flags().String("cc", "", "CC email address")
	cmd.Flags().String("bcc", "", "BCC email address")
	cmd.Flags().String("language-code", "", "Language/locale code (e.g. en-US)")
	cmd.Flags().String("email-format", "", `"styled" or "plain"`)
	cmd.Flags().String("lmx", "", "LMX markup (inline)")
	cmd.Flags().String("lmx-file", "", "Path to a file containing LMX markup")
	cmd.Flags().StringArray("contact-fallback", nil, "Contact property fallback as KEY=value (repeatable). Replaces all server-side contact fallbacks.")
	cmd.Flags().StringArray("event-fallback", nil, "Event property fallback as KEY=value (repeatable). Replaces all server-side event fallbacks.")
	cmd.Flags().StringArray("data-fallback", nil, "Transactional data-variable fallback as KEY=value (repeatable). Replaces all server-side data fallbacks.")
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
	if cmd.Flags().Changed("cc") {
		p.CCEmail, _ = cmd.Flags().GetString("cc")
		p.Set["ccEmail"] = true
	}
	if cmd.Flags().Changed("bcc") {
		p.BCCEmail, _ = cmd.Flags().GetString("bcc")
		p.Set["bccEmail"] = true
	}
	if cmd.Flags().Changed("language-code") {
		p.LanguageCode, _ = cmd.Flags().GetString("language-code")
		p.Set["languageCode"] = true
	}
	if cmd.Flags().Changed("email-format") {
		v, _ := cmd.Flags().GetString("email-format")
		if v != loops.EmailFormatStyled && v != loops.EmailFormatPlain {
			return p, fmt.Errorf("--email-format must be %q or %q", loops.EmailFormatStyled, loops.EmailFormatPlain)
		}
		p.EmailFormat = v
		p.Set["emailFormat"] = true
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
	if cmd.Flags().Changed("contact-fallback") {
		pairs, _ := cmd.Flags().GetStringArray("contact-fallback")
		m, err := parseFallbacks("contact-fallback", pairs)
		if err != nil {
			return p, err
		}
		p.ContactPropertiesFallbacks = m
		p.Set["contactPropertiesFallbacks"] = true
	}
	if cmd.Flags().Changed("event-fallback") {
		pairs, _ := cmd.Flags().GetStringArray("event-fallback")
		m, err := parseFallbacks("event-fallback", pairs)
		if err != nil {
			return p, err
		}
		p.EventPropertiesFallbacks = m
		p.Set["eventPropertiesFallbacks"] = true
	}
	if cmd.Flags().Changed("data-fallback") {
		pairs, _ := cmd.Flags().GetStringArray("data-fallback")
		m, err := parseFallbacks("data-fallback", pairs)
		if err != nil {
			return p, err
		}
		p.DataVariablesFallbacks = m
		p.Set["dataVariablesFallbacks"] = true
	}
	return p, nil
}

// parseFallbacks parses repeated KEY=value pairs into a map[string]*string
// suitable for the SDK's *Fallbacks fields. Values are taken literally; there
// is no support for setting a key to a nil pointer (server-side clear) via the
// CLI in this round.
func parseFallbacks(flag string, pairs []string) (map[string]*string, error) {
	m := make(map[string]*string, len(pairs))
	for _, pair := range pairs {
		idx := strings.IndexByte(pair, '=')
		if idx < 0 {
			return nil, fmt.Errorf("--%s %q: expected KEY=value", flag, pair)
		}
		key := pair[:idx]
		val := pair[idx+1:]
		m[key] = &val
	}
	return m, nil
}

// parseStringPropertyMap parses repeated KEY=value pairs into a map[string]string.
func parseStringPropertyMap(flag string, pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		idx := strings.IndexByte(pair, '=')
		if idx < 0 {
			return nil, fmt.Errorf("--%s %q: expected KEY=value", flag, pair)
		}
		m[pair[:idx]] = pair[idx+1:]
	}
	return m, nil
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
	Use:   "email-messages",
	Short: "Manage email messages",
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
				Subject:                    params.Subject,
				PreviewText:                params.PreviewText,
				FromName:                   params.FromName,
				FromEmail:                  params.FromEmail,
				ReplyToEmail:               params.ReplyToEmail,
				CCEmail:                    params.CCEmail,
				BCCEmail:                   params.BCCEmail,
				LanguageCode:               params.LanguageCode,
				EmailFormat:                params.EmailFormat,
				LMX:                        params.LMX,
				ContactPropertiesFallbacks: params.ContactPropertiesFallbacks,
				EventPropertiesFallbacks:   params.EventPropertiesFallbacks,
				DataVariablesFallbacks:     params.DataVariablesFallbacks,
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

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (emailMessageId: %s, contentRevisionId: %s)\n", msg.ID, deref(msg.ContentRevisionID))
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
	t.Row("emailMessageId", msg.ID)
	t.Row("campaignId", deref(msg.CampaignID))
	t.Row("transactionalId", deref(msg.TransactionalID))
	t.Row("subject", msg.Subject)
	t.Row("previewText", msg.PreviewText)
	t.Row("fromName", msg.FromName)
	t.Row("fromEmail", msg.FromEmail)
	t.Row("replyToEmail", msg.ReplyToEmail)
	if msg.CCEmail != "" {
		t.Row("ccEmail", msg.CCEmail)
	}
	if msg.BCCEmail != "" {
		t.Row("bccEmail", msg.BCCEmail)
	}
	if msg.LanguageCode != "" {
		t.Row("languageCode", msg.LanguageCode)
	}
	if msg.EmailFormat != "" {
		t.Row("emailFormat", msg.EmailFormat)
	}
	if s := formatStringMap(msg.ContactPropertiesFallbacks); s != "" {
		t.Row("contactPropertiesFallbacks", s)
	}
	if s := formatStringMap(msg.EventPropertiesFallbacks); s != "" {
		t.Row("eventPropertiesFallbacks", s)
	}
	if s := formatStringMap(msg.DataVariablesFallbacks); s != "" {
		t.Row("dataVariablesFallbacks", s)
	}
	t.Row("contentRevisionId", deref(msg.ContentRevisionID))
	t.Row("updatedAt", msg.UpdatedAt)
	if err := t.Render(); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout())
	return renderLMX(cmd.OutOrStdout(), msg.LMX)
}

// formatStringMap renders a map as "k1=v1, k2=v2" with deterministic key order.
// Returns an empty string for nil or empty maps so callers can skip the row.
func formatStringMap(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + m[k]
	}
	return strings.Join(parts, ", ")
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

func runEmailMessagesPreview(cfg *config.Config, id string, req loops.EmailMessagePreviewRequest) (*loops.EmailMessagePreviewResponse, error) {
	return newAPIClient(cfg).PreviewEmailMessage(id, req)
}

var emailMessagesPreviewCmd = &cobra.Command{
	Use:   "preview <id>",
	Short: "Send a preview of an email message to one or more addresses",
	Long: "Sends a test preview of the email message to the addresses in --email.\n" +
		"Variable fields accepted depend on the parent type:\n" +
		"  - campaign previews accept --contact-prop\n" +
		"  - workflow previews accept --contact-prop and --event-prop\n" +
		"  - transactional previews accept --var / --json-vars",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emails, _ := cmd.Flags().GetStringArray("email")
		contactPairs, _ := cmd.Flags().GetStringArray("contact-prop")
		eventPairs, _ := cmd.Flags().GetStringArray("event-prop")
		varPairs, _ := cmd.Flags().GetStringArray("var")
		jsonFile, _ := cmd.Flags().GetString("json-vars")

		contactProps, err := parseStringPropertyMap("contact-prop", contactPairs)
		if err != nil {
			return err
		}
		eventProps, err := parseStringPropertyMap("event-prop", eventPairs)
		if err != nil {
			return err
		}
		dataVars, err := parseDataVars(varPairs, jsonFile)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		resp, err := runEmailMessagesPreview(cfg, args[0], loops.EmailMessagePreviewRequest{
			Emails:            emails,
			ContactProperties: contactProps,
			EventProperties:   eventProps,
			DataVariables:     dataVars,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), resp)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Preview sent. (id: %s)\n", resp.ID)
		return nil
	},
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
	emailMessagesUpdateCmd.MarkFlagsOneRequired(
		"subject", "preview-text", "from-name", "from-email", "reply-to",
		"cc", "bcc", "language-code", "email-format",
		"lmx", "lmx-file",
		"contact-fallback", "event-fallback", "data-fallback",
	)
	emailMessagesCmd.AddCommand(emailMessagesUpdateCmd)

	emailMessagesPreviewCmd.Flags().StringArray("email", nil, "Recipient email address (repeatable; at least one required)")
	emailMessagesPreviewCmd.MarkFlagRequired("email")
	emailMessagesPreviewCmd.Flags().StringArray("contact-prop", nil, "Contact property as KEY=value (repeatable)")
	emailMessagesPreviewCmd.Flags().StringArray("event-prop", nil, "Event property as KEY=value (repeatable)")
	emailMessagesPreviewCmd.Flags().StringArrayP("var", "v", nil, "Data variable as KEY=value (repeatable; transactional previews only)")
	emailMessagesPreviewCmd.Flags().StringP("json-vars", "j", "", "Path to a JSON file of data variables")
	emailMessagesCmd.AddCommand(emailMessagesPreviewCmd)

	rootCmd.AddCommand(emailMessagesCmd)
}
