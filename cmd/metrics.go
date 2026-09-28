package cmd

import (
	"strconv"

	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func printEmailMetrics(cmd *cobra.Command, m *loops.EmailMetrics) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("sends", strconv.Itoa(m.Sends))
	t.Row("opens", strconv.Itoa(m.Opens))
	t.Row("clicks", strconv.Itoa(m.Clicks))
	t.Row("unsubscribes", strconv.Itoa(m.Unsubscribes))
	t.Row("spamReports", strconv.Itoa(m.SpamReports))
	t.Row("hardBounces", strconv.Itoa(m.HardBounces))
	t.Row("softBounces", strconv.Itoa(m.SoftBounces))
	return t.Render()
}

func printTransactionalMetrics(cmd *cobra.Command, m *loops.TransactionalMetrics) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("sends", strconv.Itoa(m.Sends))
	t.Row("deliveries", strconv.Itoa(m.Deliveries))
	t.Row("spamReports", strconv.Itoa(m.SpamReports))
	t.Row("hardBounces", strconv.Itoa(m.HardBounces))
	t.Row("softBounces", strconv.Itoa(m.SoftBounces))
	return t.Render()
}
