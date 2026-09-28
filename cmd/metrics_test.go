package cmd

import (
	"strings"
	"testing"

	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func fieldValueRows(out string) map[string]string {
	rows := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 {
			rows[f[0]] = f[1]
		}
	}
	return rows
}

func TestPrintEmailMetrics(t *testing.T) {
	var out strings.Builder
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	err := printEmailMetrics(cmd, &loops.EmailMetrics{
		Sends:        4210,
		Opens:        1922,
		Clicks:       301,
		Unsubscribes: 12,
		SpamReports:  1,
		HardBounces:  9,
		SoftBounces:  13,
	})
	if err != nil {
		t.Fatalf("printEmailMetrics: %v", err)
	}

	want := map[string]string{
		"sends":        "4210",
		"opens":        "1922",
		"clicks":       "301",
		"unsubscribes": "12",
		"spamReports":  "1",
		"hardBounces":  "9",
		"softBounces":  "13",
	}
	got := fieldValueRows(out.String())
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q\n%s", k, got[k], v, out.String())
		}
	}
}

func TestPrintTransactionalMetrics(t *testing.T) {
	var out strings.Builder
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	err := printTransactionalMetrics(cmd, &loops.TransactionalMetrics{
		Sends:       4210,
		Deliveries:  4175,
		SpamReports: 1,
		HardBounces: 9,
		SoftBounces: 13,
	})
	if err != nil {
		t.Fatalf("printTransactionalMetrics: %v", err)
	}

	want := map[string]string{
		"sends":       "4210",
		"deliveries":  "4175",
		"spamReports": "1",
		"hardBounces": "9",
		"softBounces": "13",
	}
	got := fieldValueRows(out.String())
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q\n%s", k, got[k], v, out.String())
		}
	}
	if _, ok := got["opens"]; ok {
		t.Errorf("transactional metrics printed an opens row:\n%s", out.String())
	}
}
