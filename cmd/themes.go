package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

// themeFieldParams holds the fields shared by `themes create` and
// `themes update`. Name is empty when the flag is unset; Styles is nil unless
// --styles-file is provided.
type themeFieldParams struct {
	Name   string
	Styles *loops.ThemeStyles
}

func addThemeFieldFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "Theme name")
	cmd.Flags().String("styles-file", "", "Path to a JSON file with theme styles")
}

func themeFieldParamsFromCmd(cmd *cobra.Command) (themeFieldParams, error) {
	var p themeFieldParams
	if cmd.Flags().Changed("name") {
		p.Name, _ = cmd.Flags().GetString("name")
	}
	if cmd.Flags().Changed("styles-file") {
		path, _ := cmd.Flags().GetString("styles-file")
		if path == "" {
			return p, fmt.Errorf("--styles-file requires a value")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return p, fmt.Errorf("read --styles-file: %w", err)
		}
		var styles loops.ThemeStyles
		if err := json.Unmarshal(data, &styles); err != nil {
			return p, fmt.Errorf("parse --styles-file: %w", err)
		}
		p.Styles = &styles
	}
	return p, nil
}

func runThemesGet(cfg *config.Config, id string) (*loops.Theme, error) {
	return newAPIClient(cfg).GetTheme(id)
}

func runThemesCreate(cfg *config.Config, req loops.CreateThemeRequest) (*loops.Theme, error) {
	return newAPIClient(cfg).CreateTheme(req)
}

func runThemesUpdate(cfg *config.Config, id string, req loops.UpdateThemeRequest) (*loops.Theme, error) {
	return newAPIClient(cfg).UpdateTheme(id, req)
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
				th.ID,
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

		return printTheme(cmd, th)
	},
}

var themesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a theme",
	RunE: func(cmd *cobra.Command, args []string) error {
		params, err := themeFieldParamsFromCmd(cmd)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		th, err := runThemesCreate(cfg, loops.CreateThemeRequest{
			Name:   params.Name,
			Styles: params.Styles,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), th)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created. (id: %s)\n\n", th.ID)
		return printTheme(cmd, th)
	},
}

var themesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a theme",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		params, err := themeFieldParamsFromCmd(cmd)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		th, err := runThemesUpdate(cfg, args[0], loops.UpdateThemeRequest{
			Name:   params.Name,
			Styles: params.Styles,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), th)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (id: %s)\n\n", th.ID)
		return printTheme(cmd, th)
	},
}

func printTheme(cmd *cobra.Command, th *loops.Theme) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("themeId", th.ID)
	t.Row("name", th.Name)
	t.Row("isDefault", strconv.FormatBool(th.IsDefault))
	t.Row("createdAt", th.CreatedAt)
	t.Row("updatedAt", th.UpdatedAt)
	if err := t.Render(); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout())
	return printThemeStyles(cmd, th.Styles)
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

	addThemeFieldFlags(themesCreateCmd)
	themesCreateCmd.MarkFlagRequired("name")
	themesCmd.AddCommand(themesCreateCmd)

	addThemeFieldFlags(themesUpdateCmd)
	themesUpdateCmd.MarkFlagsOneRequired("name", "styles-file")
	themesCmd.AddCommand(themesUpdateCmd)

	rootCmd.AddCommand(themesCmd)
}
