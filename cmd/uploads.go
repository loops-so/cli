package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func runUploadsCreate(cfg *config.Config, path, contentType string) (*loops.CompleteUploadResponse, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	if contentType == "" {
		var sniff [512]byte
		n, err := f.Read(sniff[:])
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", path, err)
		}
		contentType = http.DetectContentType(sniff[:n])
		if _, err := f.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("seek %q: %w", path, err)
		}
	}

	return newAPIClient(cfg).Upload(loops.UploadRequest{
		ContentType:   contentType,
		ContentLength: info.Size(),
		Body:          f,
	})
}

var uploadsCmd = &cobra.Command{
	Use:   "uploads",
	Short: "Manage uploaded email assets",
}

var uploadsCreateCmd = &cobra.Command{
	Use:   "create <path>",
	Short: "Upload a file as an email asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contentType, _ := cmd.Flags().GetString("content-type")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		result, err := runUploadsCreate(cfg, args[0], contentType)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), result)
		}

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("emailAssetId", result.EmailAssetID)
		t.Row("finalUrl", result.FinalURL)
		return t.Render()
	},
}

func init() {
	uploadsCreateCmd.Flags().String("content-type", "", "MIME type to use for the upload (default: sniffed from file contents)")

	uploadsCmd.AddCommand(uploadsCreateCmd)
	rootCmd.AddCommand(uploadsCmd)
}
