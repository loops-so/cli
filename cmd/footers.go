package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func runFootersList(cfg *config.Config, params loops.PaginationParams) ([]loops.Footer, error) {
	client := newAPIClient(cfg)
	if params.Cursor != "" {
		footers, _, err := client.ListFooters(params)
		return footers, err
	}
	return loops.Paginate(func(cursor string) ([]loops.Footer, *loops.Pagination, error) {
		return client.ListFooters(loops.PaginationParams{
			PerPage: params.PerPage,
			Cursor:  cursor,
		})
	})
}

func runFootersGet(cfg *config.Config, id string) (*loops.FooterWithVariants, error) {
	return newAPIClient(cfg).GetFooter(id)
}

func runFootersCreate(cfg *config.Config, req loops.CreateFooterRequest) (*loops.FooterWithVariants, error) {
	return newAPIClient(cfg).CreateFooter(req)
}

func runFootersUpdate(cfg *config.Config, id string, req loops.UpdateFooterRequest) (*loops.FooterWithVariants, error) {
	return newAPIClient(cfg).UpdateFooter(id, req)
}

func runFootersDelete(cfg *config.Config, id string) error {
	return newAPIClient(cfg).DeleteFooter(id)
}

func runFootersSetDefault(cfg *config.Config, id string, req loops.SetDefaultFooterRequest) (*loops.SetDefaultFooterResponse, error) {
	return newAPIClient(cfg).SetDefaultFooter(id, req)
}

// runFootersVariantGet returns the variant of the footer identified by
// footerID. A footer has at most one variant, so the variant commands take the
// footer ID only. A footer with more than one variant is an error, because the
// variant to use is then not known.
func runFootersVariantGet(cfg *config.Config, footerID string) (*loops.FooterVariant, error) {
	f, err := newAPIClient(cfg).GetFooter(footerID)
	if err != nil {
		return nil, err
	}
	switch len(f.Variants) {
	case 0:
		return nil, fmt.Errorf("footer %s has no variant", footerID)
	case 1:
		return &f.Variants[0], nil
	}
	ids := make([]string, len(f.Variants))
	for i, v := range f.Variants {
		ids[i] = v.ID
	}
	return nil, fmt.Errorf("footer %s has %d variants (%s); this version of the CLI supports one variant per footer",
		footerID, len(ids), strings.Join(ids, ", "))
}

func runFootersVariantUpdate(cfg *config.Config, footerID, variantID string, req loops.UpdateFooterVariantRequest) (*loops.FooterVariant, error) {
	return newAPIClient(cfg).UpdateFooterVariant(footerID, variantID, req)
}

func runFootersVariantPublish(cfg *config.Config, footerID, variantID string, req loops.PublishFooterVariantRequest) (*loops.FooterVariant, error) {
	return newAPIClient(cfg).PublishFooterVariant(footerID, variantID, req)
}

func addFooterVariantRevisionFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("expected-revision-id", "r", "", "Last-seen contentRevisionId. Get this from a prior 'footers variant get'. Mutually exclusive with --force.")
	cmd.Flags().BoolP("force", "f", false, "Use the current revision (overwrites any concurrent edits). Mutually exclusive with --expected-revision-id.")
	cmd.MarkFlagsMutuallyExclusive("expected-revision-id", "force")
	cmd.MarkFlagsOneRequired("expected-revision-id", "force")
}

// footerVariantTargetFromCmd resolves the variant ID of the footer and the
// revision ID to send: the current revision with --force, otherwise the value
// of --expected-revision-id.
func footerVariantTargetFromCmd(cmd *cobra.Command, cfg *config.Config, footerID string) (variantID, revisionID string, err error) {
	v, err := runFootersVariantGet(cfg, footerID)
	if err != nil {
		return "", "", err
	}
	if force, _ := cmd.Flags().GetBool("force"); force {
		return v.ID, v.ContentRevisionID, nil
	}
	revisionID, _ = cmd.Flags().GetString("expected-revision-id")
	return v.ID, revisionID, nil
}

var footersCmd = &cobra.Command{
	Use:   "footers",
	Short: "Manage footers",
}

var footersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List footers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePickFlags(cmd); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		footers, err := runFootersList(cfg, paginationParams(cmd))
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if footers == nil {
				footers = []loops.Footer{}
			}
			return printJSON(cmd.OutOrStdout(), footers)
		}

		if len(footers) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No footers found.")
			return nil
		}

		headers := []string{"ID", "NAME", "DEFAULT", "PLACEMENT", "UPDATED"}
		rows := make([][]string, 0, len(footers))
		for _, f := range footers {
			rows = append(rows, []string{
				f.ID,
				f.Name,
				strconv.FormatBool(f.IsDefault),
				f.Placement,
				f.UpdatedAt,
			})
		}

		if isPicking(cmd) {
			return runPicker(headers, rows, []pickBinding{
				copyColumnBinding("enter", "copy id", "footer ID", rows, 0, cmd.OutOrStdout()),
			})
		}

		t := newStyledTable(cmd.OutOrStdout(), headers...)
		for _, r := range rows {
			t.Row(r...)
		}
		return t.Render()
	},
}

var footersGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a footer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		f, err := runFootersGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), f)
		}

		return printFooter(cmd, f)
	},
}

var footersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a footer",
	Long: "Creates a footer with an unpublished draft. Emails cannot use the footer until its\n" +
		"content is published with 'footers variant publish'.",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		f, err := runFootersCreate(cfg, loops.CreateFooterRequest{Name: name})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), f)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created. (id: %s)\n\n", f.ID)
		return printFooter(cmd, f)
	},
}

var footersUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update the name or placement of a footer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		placement, _ := cmd.Flags().GetString("placement")
		if cmd.Flags().Changed("placement") && placement != loops.FooterPlacementInside && placement != loops.FooterPlacementBelow {
			return fmt.Errorf("--placement must be %q or %q", loops.FooterPlacementInside, loops.FooterPlacementBelow)
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		f, err := runFootersUpdate(cfg, args[0], loops.UpdateFooterRequest{
			Name:      name,
			Placement: placement,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), f)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (id: %s)\n\n", f.ID)
		return printFooter(cmd, f)
	},
}

var footersDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a footer",
	Long:  "Deletes a footer. The API rejects the delete when the footer is the default footer or when emails use it.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if err := runFootersDelete(cfg, args[0]); err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), Result{Success: true})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Deleted.")
		return nil
	},
}

var footersSetDefaultCmd = &cobra.Command{
	Use:   "set-default <id>",
	Short: "Make a published footer the default footer",
	Long: "Makes a published footer the default footer. New campaign and workflow emails use the default footer.\n" +
		"--existing-emails selects which existing campaign and workflow emails switch to the footer:\n" +
		"  - previousDefault: the emails that use the previous default footer\n" +
		"  - all: all emails (asks for confirmation; skip the prompt with --yes)\n" +
		"  - none: no emails",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		existing, _ := cmd.Flags().GetString("existing-emails")
		switch existing {
		case loops.FooterExistingEmailsPreviousDefault, loops.FooterExistingEmailsNone:
		case loops.FooterExistingEmailsAll:
			prompt := fmt.Sprintf("Switch all existing campaign and workflow emails to footer %s?", args[0])
			if err := confirmAction(cmd, prompt); err != nil {
				return err
			}
		default:
			return fmt.Errorf("--existing-emails must be %q, %q, or %q",
				loops.FooterExistingEmailsPreviousDefault, loops.FooterExistingEmailsAll, loops.FooterExistingEmailsNone)
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		resp, err := runFootersSetDefault(cfg, args[0], loops.SetDefaultFooterRequest{ExistingEmails: existing})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), resp)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Default footer set. (id: %s, updatedEmailCount: %d)\n\n", resp.ID, resp.UpdatedEmailCount)
		return printFooter(cmd, &resp.FooterWithVariants)
	},
}

var footersVariantCmd = &cobra.Command{
	Use:   "variant",
	Short: "Manage the content of a footer",
}

var footersVariantGetCmd = &cobra.Command{
	Use:   "get <footer-id>",
	Short: "Get the draft and published content of a footer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		v, err := runFootersVariantGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), v)
		}

		return printFooterVariant(cmd, v)
	},
}

var footersVariantUpdateCmd = &cobra.Command{
	Use:   "update <footer-id>",
	Short: "Update the draft content of a footer",
	Long: "Updates the draft LMX content of a footer. Emails keep the published content until the\n" +
		"draft is published with 'footers variant publish'.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		lmx, _, err := lmxFromCmd(cmd)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		variantID, revisionID, err := footerVariantTargetFromCmd(cmd, cfg, args[0])
		if err != nil {
			return err
		}

		v, err := runFootersVariantUpdate(cfg, args[0], variantID, loops.UpdateFooterVariantRequest{
			ExpectedRevisionID: revisionID,
			LMX:                lmx,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), v)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (footerId: %s, contentRevisionId: %s)\n\n", v.FooterID, v.ContentRevisionID)
		return printFooterVariant(cmd, v)
	},
}

var footersVariantPublishCmd = &cobra.Command{
	Use:   "publish <footer-id>",
	Short: "Publish the draft content of a footer",
	Long: "Publishes the draft LMX content of a footer. Emails that use the footer use the new content at once.\n" +
		"The draft must have {system.company_name} and {system.company_address} in the text and\n" +
		"{system.unsubscribe_link} as the URL of a link or button.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		variantID, revisionID, err := footerVariantTargetFromCmd(cmd, cfg, args[0])
		if err != nil {
			return err
		}

		v, err := runFootersVariantPublish(cfg, args[0], variantID, loops.PublishFooterVariantRequest{
			ExpectedRevisionID: revisionID,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), v)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Published. (footerId: %s, contentRevisionId: %s)\n\n", v.FooterID, v.ContentRevisionID)
		return printFooterVariant(cmd, v)
	},
}

func printFooter(cmd *cobra.Command, f *loops.FooterWithVariants) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("footerId", f.ID)
	t.Row("name", f.Name)
	t.Row("isDefault", strconv.FormatBool(f.IsDefault))
	t.Row("placement", f.Placement)
	t.Row("createdAt", f.CreatedAt)
	t.Row("updatedAt", f.UpdatedAt)
	if err := t.Render(); err != nil {
		return err
	}

	for i := range f.Variants {
		fmt.Fprintln(cmd.OutOrStdout())
		if err := printFooterVariant(cmd, &f.Variants[i]); err != nil {
			return err
		}
	}
	return nil
}

// printFooterVariant renders a variant as a field table followed by its
// highlighted draft LMX. When the published LMX is different from the draft,
// the two are rendered under "Draft:" and "Published:" headings.
func printFooterVariant(cmd *cobra.Command, v *loops.FooterVariant) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("variantId", v.ID)
	t.Row("footerId", v.FooterID)
	t.Row("contentRevisionId", v.ContentRevisionID)
	t.Row("publishedAt", dashIfEmpty(deref(v.PublishedAt)))
	t.Row("unpublishedChanges", strconv.FormatBool(v.PublishedLMX == nil || *v.PublishedLMX != v.DraftLMX))
	t.Row("updatedAt", v.UpdatedAt)
	if err := t.Render(); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	if v.PublishedLMX == nil || *v.PublishedLMX == v.DraftLMX {
		return renderLMX(out, v.DraftLMX)
	}

	fmt.Fprintln(out, "Draft:")
	if err := renderLMX(out, v.DraftLMX); err != nil {
		return err
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Published:")
	return renderLMX(out, *v.PublishedLMX)
}

func init() {
	addPaginationFlags(footersListCmd)
	addPickFlag(footersListCmd)
	footersCmd.AddCommand(footersListCmd)
	footersCmd.AddCommand(footersGetCmd)

	footersCreateCmd.Flags().StringP("name", "n", "", "Footer name")
	footersCmd.AddCommand(footersCreateCmd)

	footersUpdateCmd.Flags().StringP("name", "n", "", "Footer name")
	footersUpdateCmd.Flags().String("placement", "", `"inside" or "below" the email body`)
	footersUpdateCmd.MarkFlagsOneRequired("name", "placement")
	footersCmd.AddCommand(footersUpdateCmd)

	footersCmd.AddCommand(footersDeleteCmd)

	footersSetDefaultCmd.Flags().String("existing-emails", "", `Existing emails that switch to the footer: "previousDefault", "all", or "none"`)
	footersSetDefaultCmd.MarkFlagRequired("existing-emails")
	addYesFlag(footersSetDefaultCmd)
	footersCmd.AddCommand(footersSetDefaultCmd)

	footersVariantCmd.AddCommand(footersVariantGetCmd)

	footersVariantUpdateCmd.Flags().String("lmx", "", "LMX markup (inline)")
	footersVariantUpdateCmd.Flags().String("lmx-file", "", "Path to a file containing LMX markup")
	footersVariantUpdateCmd.MarkFlagsMutuallyExclusive("lmx", "lmx-file")
	footersVariantUpdateCmd.MarkFlagsOneRequired("lmx", "lmx-file")
	addFooterVariantRevisionFlags(footersVariantUpdateCmd)
	footersVariantCmd.AddCommand(footersVariantUpdateCmd)

	addFooterVariantRevisionFlags(footersVariantPublishCmd)
	footersVariantCmd.AddCommand(footersVariantPublishCmd)

	footersCmd.AddCommand(footersVariantCmd)

	rootCmd.AddCommand(footersCmd)
}
