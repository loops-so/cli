package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

// openBrowser launches the default browser for url. Tests replace it.
var openBrowser = func(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Run()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	default:
		return exec.Command("xdg-open", url).Run()
	}
}

func addWebFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("web", "w", false, "Open in the Loops web app instead of printing details")
}

func isWeb(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("web")
	return v
}

func openResource(cmd *cobra.Command, kind, id, url string) error {
	if url == "" {
		return fmt.Errorf("no URL returned for %s %s", kind, id)
	}
	if err := openBrowser(url); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}
	if isJSONOutput() {
		return printJSON(cmd.OutOrStdout(), Result{Success: true, URL: url})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Opening %s in your browser.\n", url)
	return nil
}
