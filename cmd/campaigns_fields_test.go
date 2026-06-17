package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

// newCampaignFieldCmd returns a bare cobra.Command wired up with the campaign
// field flags so tests can exercise campaignFieldParamsFromCmd in isolation.
func newCampaignFieldCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	addCampaignFieldFlags(cmd)
	return cmd
}

func TestCampaignFieldParamsFromCmd(t *testing.T) {
	t.Run("no flags returns empty Set", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags(nil); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(p.Set) != 0 {
			t.Errorf("Set = %v, want empty", p.Set)
		}
	})

	t.Run("name flag sets only name key", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--name", "Spring"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Name != "Spring" {
			t.Errorf("Name = %q, want Spring", p.Name)
		}
		if !p.Set["name"] || len(p.Set) != 1 {
			t.Errorf("Set = %v, want {name:true}", p.Set)
		}
	})

	t.Run("mailing-list-id is set as pointer", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--mailing-list-id", "ml_1"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.MailingListID == nil || *p.MailingListID != "ml_1" {
			t.Errorf("MailingListID = %v, want pointer to ml_1", p.MailingListID)
		}
		if !p.Set["mailingListId"] {
			t.Errorf("Set[mailingListId] not true: %v", p.Set)
		}
	})

	t.Run("schedule-now produces method now", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--schedule-now"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Scheduling == nil || p.Scheduling.Method != loops.CampaignSchedulingMethodNow {
			t.Errorf("Scheduling = %+v, want {Method: now}", p.Scheduling)
		}
		if !p.Set["scheduling"] {
			t.Errorf("Set[scheduling] not true: %v", p.Set)
		}
	})

	t.Run("schedule-at produces method schedule with timestamp", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--schedule-at", "2026-07-01T12:00:00Z"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Scheduling == nil {
			t.Fatal("Scheduling nil")
		}
		if p.Scheduling.Method != loops.CampaignSchedulingMethodSchedule {
			t.Errorf("Method = %q, want schedule", p.Scheduling.Method)
		}
		if p.Scheduling.Timestamp != "2026-07-01T12:00:00Z" {
			t.Errorf("Timestamp = %q", p.Scheduling.Timestamp)
		}
	})

	t.Run("audience-filter-file parses JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "filter.json")
		filter := `{
			"match": "all",
			"conditions": [
				{"type": "property", "key": "plan", "operator": "equals", "value": "pro"},
				{"type": "optIn", "status": "accepted"}
			]
		}`
		if err := os.WriteFile(path, []byte(filter), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--audience-filter-file", path}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.AudienceFilter == nil {
			t.Fatal("AudienceFilter nil")
		}
		if p.AudienceFilter.Match != "all" {
			t.Errorf("Match = %q, want all", p.AudienceFilter.Match)
		}
		if len(p.AudienceFilter.Conditions) != 2 {
			t.Fatalf("Conditions len = %d, want 2", len(p.AudienceFilter.Conditions))
		}
		if p.AudienceFilter.Conditions[0].Type != loops.AudienceConditionTypeProperty {
			t.Errorf("Conditions[0].Type = %q", p.AudienceFilter.Conditions[0].Type)
		}
		if p.AudienceFilter.Conditions[1].Type != loops.AudienceConditionTypeOptIn {
			t.Errorf("Conditions[1].Type = %q", p.AudienceFilter.Conditions[1].Type)
		}
	})

	t.Run("audience-filter-file missing returns error", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--audience-filter-file", "/no/such/file.json"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		if _, err := campaignFieldParamsFromCmd(cmd); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
