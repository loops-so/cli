package cmd

import (
	"os"
	"path/filepath"
	"strings"
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

	t.Run("audience-filter parses inline JSON", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		filter := `{"match":"any","conditions":[{"type":"property","key":"plan","operator":"equals","value":"pro"}]}`
		if err := cmd.ParseFlags([]string{"--audience-filter", filter}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.AudienceFilter == nil {
			t.Fatal("AudienceFilter nil")
		}
		if p.AudienceFilter.Match != "any" {
			t.Errorf("Match = %q, want any", p.AudienceFilter.Match)
		}
		if len(p.AudienceFilter.Conditions) != 1 {
			t.Fatalf("Conditions len = %d, want 1", len(p.AudienceFilter.Conditions))
		}
		if p.AudienceFilter.Conditions[0].Type != loops.AudienceConditionTypeProperty {
			t.Errorf("Conditions[0].Type = %q", p.AudienceFilter.Conditions[0].Type)
		}
		if !p.Set["audienceFilter"] {
			t.Error(`Set["audienceFilter"] = false, want true`)
		}
	})

	t.Run("audience-filter invalid JSON returns error", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--audience-filter", "{not json"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		_, err := campaignFieldParamsFromCmd(cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "parse --audience-filter:") {
			t.Errorf("error = %q, want a %q parse error", err.Error(), "parse --audience-filter:")
		}
	})

	t.Run("audience-filter-file invalid JSON returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "filter.json")
		if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--audience-filter-file", path}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		_, err := campaignFieldParamsFromCmd(cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "parse --audience-filter-file:") {
			t.Errorf("error = %q, want a %q parse error", err.Error(), "parse --audience-filter-file:")
		}
	})

	t.Run(`audience-filter "null" sentinel clears the filter`, func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--audience-filter", "null"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.AudienceFilter != nil {
			t.Errorf("AudienceFilter = %v, want nil", p.AudienceFilter)
		}
		if !p.Set["audienceFilter"] {
			t.Error(`Set["audienceFilter"] = false, want true`)
		}
	})

	t.Run("audience-filter and audience-filter-file together return error", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--audience-filter", "{}", "--audience-filter-file", "f.json"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		_, err := campaignFieldParamsFromCmd(cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "mutually exclusive") {
			t.Errorf("error = %q, want it to mention mutually exclusive", err.Error())
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

	t.Run(`mailing-list-id "null" sentinel clears the field`, func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--mailing-list-id", "null"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.MailingListID != nil {
			t.Errorf("MailingListID = %v, want nil", p.MailingListID)
		}
		if !p.Set["mailingListId"] {
			t.Error(`Set["mailingListId"] = false, want true`)
		}
	})

	t.Run(`audience-segment-id "null" sentinel clears the field`, func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--audience-segment-id", "null"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		p, err := campaignFieldParamsFromCmd(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.AudienceSegmentID != nil {
			t.Errorf("AudienceSegmentID = %v, want nil", p.AudienceSegmentID)
		}
		if !p.Set["audienceSegmentId"] {
			t.Error(`Set["audienceSegmentId"] = false, want true`)
		}
	})

	t.Run(`audience-filter-file "null" is treated as a path, not a clear`, func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--audience-filter-file", "null"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		_, err := campaignFieldParamsFromCmd(cmd)
		if err == nil {
			t.Fatal("expected error reading file \"null\", got nil")
		}
		if !strings.Contains(err.Error(), "read --audience-filter-file") {
			t.Errorf("error = %q, want a file-read error", err.Error())
		}
	})

	t.Run("empty value on nullable string flag is rejected", func(t *testing.T) {
		cases := []string{"mailing-list-id", "audience-segment-id", "audience-filter"}
		for _, flag := range cases {
			t.Run(flag, func(t *testing.T) {
				cmd := newCampaignFieldCmd()
				if err := cmd.ParseFlags([]string{"--" + flag, ""}); err != nil {
					t.Fatalf("ParseFlags: %v", err)
				}
				_, err := campaignFieldParamsFromCmd(cmd)
				if err == nil {
					t.Fatalf("--%s with empty value: expected error, got nil", flag)
				}
				if !strings.Contains(err.Error(), `"null"`) {
					t.Errorf(`--%s error = %q, want it to mention "null"`, flag, err.Error())
				}
			})
		}
	})

	t.Run("empty audience-filter-file is rejected", func(t *testing.T) {
		cmd := newCampaignFieldCmd()
		if err := cmd.ParseFlags([]string{"--audience-filter-file", ""}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		_, err := campaignFieldParamsFromCmd(cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "file path") {
			t.Errorf("error = %q, want it to mention a file path", err.Error())
		}
	})
}
