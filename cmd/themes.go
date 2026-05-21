package cmd

import (
	"fmt"
	"strconv"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func runThemesGet(cfg *config.Config, id string) (*loops.Theme, error) {
	return newAPIClient(cfg).GetTheme(id)
}

func runThemesList(cfg *config.Config, params loops.PaginationParams) ([]loops.Theme, error) {
	client := newAPIClient(cfg)
	if params.Cursor != "" {
		themes, _, err := client.ListThemes(params)
		return themes, err
	}
	return loops.Paginate(func(cursor string) ([]loops.Theme, *loops.Pagination, error) {
		return client.ListThemes(loops.PaginationParams{
			PerPage: params.PerPage,
			Cursor:  cursor,
		})
	})
}

var themesCmd = &cobra.Command{
	Use:   "themes",
	Short: "Manage themes",
}

var themesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List themes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePickFlags(cmd); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		themes, err := runThemesList(cfg, paginationParams(cmd))
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if themes == nil {
				themes = []loops.Theme{}
			}
			return printJSON(cmd.OutOrStdout(), themes)
		}

		if len(themes) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No themes found.")
			return nil
		}

		headers := []string{"ID", "NAME", "DEFAULT", "UPDATED"}
		rows := make([][]string, 0, len(themes))
		for _, th := range themes {
			rows = append(rows, []string{
				th.ThemeID,
				th.Name,
				strconv.FormatBool(th.IsDefault),
				th.UpdatedAt,
			})
		}

		if isPicking(cmd) {
			return runPicker(headers, rows, []pickBinding{
				copyColumnBinding("enter", "copy id", "theme ID", rows, 0, cmd.OutOrStdout()),
			})
		}

		t := newStyledTable(cmd.OutOrStdout(), headers...)
		for _, r := range rows {
			t.Row(r...)
		}
		return t.Render()
	},
}

var themesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a theme",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		th, err := runThemesGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), th)
		}

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("themeId", th.ThemeID)
		t.Row("name", th.Name)
		t.Row("isDefault", strconv.FormatBool(th.IsDefault))
		t.Row("createdAt", th.CreatedAt)
		t.Row("updatedAt", th.UpdatedAt)
		if err := t.Render(); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout())
		return printThemeStyles(cmd, th.Styles)
	},
}

func printThemeStyles(cmd *cobra.Command, s loops.ThemeStyles) error {
	t := newStyledTable(cmd.OutOrStdout(), "STYLE", "VALUE")
	for _, row := range themeStyleRows(s) {
		t.Row(row[0], dashIfEmpty(row[1]))
	}
	return t.Render()
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func themeStyleRows(s loops.ThemeStyles) [][2]string {
	return [][2]string{
		{"backgroundColor", s.BackgroundColor},
		{"backgroundXPadding", formatFloat(s.BackgroundXPadding)},
		{"backgroundYPadding", formatFloat(s.BackgroundYPadding)},
		{"bodyColor", s.BodyColor},
		{"bodyXPadding", formatFloat(s.BodyXPadding)},
		{"bodyYPadding", formatFloat(s.BodyYPadding)},
		{"bodyFontFamily", s.BodyFontFamily},
		{"bodyFontCategory", s.BodyFontCategory},
		{"borderColor", s.BorderColor},
		{"borderWidth", formatFloat(s.BorderWidth)},
		{"borderRadius", formatFloat(s.BorderRadius)},
		{"buttonBodyColor", s.ButtonBodyColor},
		{"buttonBodyXPadding", formatFloat(s.ButtonBodyXPadding)},
		{"buttonBodyYPadding", formatFloat(s.ButtonBodyYPadding)},
		{"buttonBorderColor", s.ButtonBorderColor},
		{"buttonBorderWidth", formatFloat(s.ButtonBorderWidth)},
		{"buttonBorderRadius", formatFloat(s.ButtonBorderRadius)},
		{"buttonTextColor", s.ButtonTextColor},
		{"buttonTextFormat", formatFloat(s.ButtonTextFormat)},
		{"buttonTextFontSize", formatFloat(s.ButtonTextFontSize)},
		{"dividerColor", s.DividerColor},
		{"dividerBorderWidth", formatFloat(s.DividerBorderWidth)},
		{"textBaseColor", s.TextBaseColor},
		{"textBaseFontSize", formatFloat(s.TextBaseFontSize)},
		{"textBaseLineHeight", formatFloat(s.TextBaseLineHeight)},
		{"textBaseLetterSpacing", formatFloat(s.TextBaseLetterSpacing)},
		{"textLinkColor", s.TextLinkColor},
		{"heading1Color", s.Heading1Color},
		{"heading1FontSize", formatFloat(s.Heading1FontSize)},
		{"heading1LineHeight", formatFloat(s.Heading1LineHeight)},
		{"heading1LetterSpacing", formatFloat(s.Heading1LetterSpacing)},
		{"heading2Color", s.Heading2Color},
		{"heading2FontSize", formatFloat(s.Heading2FontSize)},
		{"heading2LineHeight", formatFloat(s.Heading2LineHeight)},
		{"heading2LetterSpacing", formatFloat(s.Heading2LetterSpacing)},
		{"heading3Color", s.Heading3Color},
		{"heading3FontSize", formatFloat(s.Heading3FontSize)},
		{"heading3LineHeight", formatFloat(s.Heading3LineHeight)},
		{"heading3LetterSpacing", formatFloat(s.Heading3LetterSpacing)},
	}
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func init() {
	addPaginationFlags(themesListCmd)
	addPickFlag(themesListCmd)
	themesCmd.AddCommand(themesListCmd)
	themesCmd.AddCommand(themesGetCmd)
	rootCmd.AddCommand(themesCmd)
}
