package cmd

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"charm.land/fang/v2"
	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

var outputFormat outputFlag = "text"
var teamFlag string
var debugFlag bool

func newAPIClient(cfg *config.Config) *loops.Client {
	opts := []loops.Option{
		loops.WithBaseURL(cfg.EndpointURL),
		loops.WithUserAgent("loops-cli/" + version),
	}
	if cfg.Debug {
		opts = append(opts, loops.WithLogger(os.Stderr))
	}
	return loops.NewClient(cfg.APIKey, opts...)
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(teamFlag)
	if err != nil {
		return nil, err
	}
	cfg.Debug = debugFlag
	return cfg, nil
}

var rootCmd = &cobra.Command{
	Use:           "loops",
	Short:         "The official CLI for Loops (https://loops.so)",
	Long:          "The official CLI for Loops (https://loops.so)",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func jsonAwareErrorHandler(w io.Writer, styles fang.Styles, err error) {
	// fang always appends a period to error output. if err has a trailing period, strip it.
	msg := strings.TrimRight(err.Error(), ".")
	if isJSONOutput() {
		_ = printJSON(w, Result{Success: false, Message: msg})
		return
	}
	fang.DefaultErrorHandler(w, styles, errors.New(msg))
}

func Execute() {
	defer func() {
		if updateCheckDone != nil {
			select {
			case <-updateCheckDone:
			case <-time.After(500 * time.Millisecond):
			}
		}
		if updateCheckCancel != nil {
			updateCheckCancel()
		}
	}()

	wrapArgsWithNames(rootCmd)

	err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version),
		fang.WithErrorHandler(jsonAwareErrorHandler),
	)

	checkForUpdate(os.Stderr)

	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().VarP(&outputFormat, "output", "o", "Output format (text, json)")
	rootCmd.PersistentFlags().StringVarP(&teamFlag, "team", "t", "", "Team key name to use")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Print API request details before sending")
}
