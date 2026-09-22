package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// stubBrowser replaces openBrowser for the test. The returned pointer holds
// the last URL passed to it.
func stubBrowser(t *testing.T) *string {
	t.Helper()
	saved := openBrowser
	var opened string
	openBrowser = func(url string) error {
		opened = url
		return nil
	}
	t.Cleanup(func() { openBrowser = saved })
	return &opened
}

func useOutputFormat(t *testing.T, format outputFlag) {
	t.Helper()
	saved := outputFormat
	outputFormat = format
	t.Cleanup(func() { outputFormat = saved })
}

func TestOpenResource(t *testing.T) {
	const url = "https://app.loops.so/campaigns/cmp_1"

	t.Run("opens the url and prints it", func(t *testing.T) {
		useOutputFormat(t, "text")
		opened := stubBrowser(t)
		var out bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&out)

		if err := openResource(cmd, "campaign", "cmp_1", url); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *opened != url {
			t.Errorf("opened = %q, want %q", *opened, url)
		}
		if got, want := out.String(), "Opening "+url+" in your browser.\n"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("json output carries the url", func(t *testing.T) {
		useOutputFormat(t, "json")
		opened := stubBrowser(t)
		var out bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&out)

		if err := openResource(cmd, "campaign", "cmp_1", url); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *opened != url {
			t.Errorf("opened = %q, want %q", *opened, url)
		}
		var res Result
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("decode output: %v\nraw: %s", err, out.String())
		}
		if !res.Success || res.URL != url {
			t.Errorf("result = %+v, want success with url %q", res, url)
		}
	})

	t.Run("errors when the url is empty", func(t *testing.T) {
		opened := stubBrowser(t)
		err := openResource(&cobra.Command{}, "campaign", "cmp_1", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "campaign cmp_1") {
			t.Errorf("error %q should name the resource", err)
		}
		if *opened != "" {
			t.Errorf("browser opened %q, want no open", *opened)
		}
	})

	t.Run("wraps browser launch errors", func(t *testing.T) {
		saved := openBrowser
		launchErr := errors.New("no browser")
		openBrowser = func(string) error { return launchErr }
		t.Cleanup(func() { openBrowser = saved })

		err := openResource(&cobra.Command{}, "campaign", "cmp_1", url)
		if !errors.Is(err, launchErr) {
			t.Errorf("error = %v, want wrapped launch error", err)
		}
	})
}
