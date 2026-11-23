package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

const footerVariantBody = `{
	"id": "var_1",
	"footerId": "ftr_1",
	"contentRevisionId": "rev_1",
	"draftLmx": "<Paragraph>Draft</Paragraph>",
	"publishedLmx": null,
	"publishedAt": null,
	"createdAt": "2026-11-01T10:00:00Z",
	"updatedAt": "2026-11-02T10:00:00Z"
}`

const footerBody = `{
	"id": "ftr_1",
	"name": "Main",
	"isDefault": false,
	"placement": "below",
	"createdAt": "2026-11-01T10:00:00Z",
	"updatedAt": "2026-11-02T10:00:00Z",
	"variants": [` + footerVariantBody + `]
}`

func decodeBody(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var sent map[string]any
	if err := json.Unmarshal(raw, &sent); err != nil {
		t.Fatalf("decode request body: %v\nraw: %s", err, raw)
	}
	return sent
}

func TestRunFootersList(t *testing.T) {
	t.Run("returns footers", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"pagination":{"nextCursor":""},"data":[{"id":"ftr_1","name":"Main","isDefault":true,"placement":"inside","createdAt":"2026-11-01","updatedAt":"2026-11-02"}]}`)
		footers, err := runFootersList(cfg(t), loops.PaginationParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(footers) != 1 {
			t.Fatalf("expected 1 footer, got %d", len(footers))
		}
		if footers[0].ID != "ftr_1" || !footers[0].IsDefault || footers[0].Placement != "inside" {
			t.Errorf("footer = %+v", footers[0])
		}
	})

	t.Run("fetches a single page when cursor is set", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, `{"pagination":{"nextCursor":"next"},"data":[]}`)
		if _, err := runFootersList(cfg(t), loops.PaginationParams{Cursor: "cur_1", PerPage: "10"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Path != "/footers?cursor=cur_1&perPage=10" {
			t.Errorf("Path = %q", got.Path)
		}
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		serveJSON(t, http.StatusUnauthorized, `{"error":"unauthorized"}`)
		if _, err := runFootersList(cfg(t), loops.PaginationParams{}); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunFootersGet(t *testing.T) {
	t.Run("returns footer with variants", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, footerBody)
		f, err := runFootersGet(cfg(t), "ftr_1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Method != http.MethodGet || got.Path != "/footers/ftr_1" {
			t.Errorf("request = %s %s", got.Method, got.Path)
		}
		if f.Name != "Main" || len(f.Variants) != 1 || f.Variants[0].ID != "var_1" {
			t.Errorf("footer = %+v", f)
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Footer not found."}`)
		if _, err := runFootersGet(cfg(t), "ftr_missing"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunFootersCreate(t *testing.T) {
	t.Run("sends name", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, footerBody)
		f, err := runFootersCreate(cfg(t), loops.CreateFooterRequest{Name: "Main"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Method != http.MethodPost || got.Path != "/footers" {
			t.Errorf("request = %s %s", got.Method, got.Path)
		}
		if sent := decodeBody(t, got.Body); sent["name"] != "Main" {
			t.Errorf("name = %v, want Main", sent["name"])
		}
		if f.ID != "ftr_1" {
			t.Errorf("ID = %q, want ftr_1", f.ID)
		}
	})

	t.Run("omits name when unset", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusCreated, footerBody)
		if _, err := runFootersCreate(cfg(t), loops.CreateFooterRequest{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, present := decodeBody(t, got.Body)["name"]; present {
			t.Error("name should be omitted")
		}
	})
}

func TestRunFootersUpdate(t *testing.T) {
	t.Run("sends only the set fields", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, footerBody)
		_, err := runFootersUpdate(cfg(t), "ftr_1", loops.UpdateFooterRequest{Placement: loops.FooterPlacementInside})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Method != http.MethodPost || got.Path != "/footers/ftr_1" {
			t.Errorf("request = %s %s", got.Method, got.Path)
		}
		sent := decodeBody(t, got.Body)
		if sent["placement"] != "inside" {
			t.Errorf("placement = %v, want inside", sent["placement"])
		}
		if _, present := sent["name"]; present {
			t.Error("name should be omitted")
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Footer not found."}`)
		if _, err := runFootersUpdate(cfg(t), "ftr_missing", loops.UpdateFooterRequest{Name: "x"}); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunFootersDelete(t *testing.T) {
	t.Run("deletes the footer", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusNoContent, "")
		if err := runFootersDelete(cfg(t), "ftr_1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Method != http.MethodDelete || got.Path != "/footers/ftr_1" {
			t.Errorf("request = %s %s", got.Method, got.Path)
		}
	})

	t.Run("returns the api message on 409", func(t *testing.T) {
		serveJSON(t, http.StatusConflict, `{"success":false,"message":"Footer is the default footer."}`)
		err := runFootersDelete(cfg(t), "ftr_1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Footer is the default footer.") {
			t.Errorf("error = %v", err)
		}
	})
}

func TestRunFootersSetDefault(t *testing.T) {
	body := strings.Replace(footerBody, `"isDefault": false`, `"isDefault": true`, 1)
	body = strings.TrimSuffix(strings.TrimSpace(body), "}") + `, "updatedEmailCount": 7}`

	t.Run("sends existingEmails and returns the count", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, body)
		resp, err := runFootersSetDefault(cfg(t), "ftr_1", loops.SetDefaultFooterRequest{
			ExistingEmails: loops.FooterExistingEmailsPreviousDefault,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Method != http.MethodPost || got.Path != "/footers/ftr_1/set-default" {
			t.Errorf("request = %s %s", got.Method, got.Path)
		}
		if sent := decodeBody(t, got.Body); sent["existingEmails"] != "previousDefault" {
			t.Errorf("existingEmails = %v, want previousDefault", sent["existingEmails"])
		}
		if !resp.IsDefault || resp.UpdatedEmailCount != 7 {
			t.Errorf("resp = %+v", resp)
		}
	})

	t.Run("returns error on 409 unpublished footer", func(t *testing.T) {
		serveJSON(t, http.StatusConflict, `{"success":false,"message":"Footer is not published."}`)
		_, err := runFootersSetDefault(cfg(t), "ftr_1", loops.SetDefaultFooterRequest{ExistingEmails: loops.FooterExistingEmailsNone})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFootersSetDefaultCmd(t *testing.T) {
	body := strings.TrimSuffix(strings.TrimSpace(footerBody), "}") + `, "updatedEmailCount": 7}`

	run := func(t *testing.T, existing string, yes bool) (*capturedRequest, string, error) {
		t.Helper()
		useOutputFormat(t, "text")
		got := serveJSONCapture(t, http.StatusOK, body)

		var out strings.Builder
		cmd := *footersSetDefaultCmd
		cmd.SetOut(&out)
		cmd.SetErr(io.Discard)
		cmd.Flags().Set("existing-emails", existing)
		cmd.Flags().Set("yes", map[bool]string{true: "true", false: "false"}[yes])
		t.Cleanup(func() {
			cmd.Flags().Set("existing-emails", "")
			cmd.Flags().Set("yes", "false")
		})
		err := cmd.RunE(&cmd, []string{"ftr_1"})
		return got, out.String(), err
	}

	t.Run("all without --yes and without a terminal sends no request", func(t *testing.T) {
		stubStdinTerminal(t, false)
		got, _, err := run(t, "all", false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got.Method != "" {
			t.Errorf("request sent without confirmation: %s %s", got.Method, got.Path)
		}
	})

	t.Run("all with --yes sends the request", func(t *testing.T) {
		stubStdinTerminal(t, false)
		got, out, err := run(t, "all", true)
		if err != nil {
			t.Fatalf("RunE: %v", err)
		}
		if sent := decodeBody(t, got.Body); sent["existingEmails"] != "all" {
			t.Errorf("existingEmails = %v, want all", sent["existingEmails"])
		}
		if !strings.Contains(out, "updatedEmailCount: 7") {
			t.Errorf("output missing count:\n%s", out)
		}
	})

	t.Run("previousDefault does not ask for confirmation", func(t *testing.T) {
		stubStdinTerminal(t, false)
		got, _, err := run(t, "previousDefault", false)
		if err != nil {
			t.Fatalf("RunE: %v", err)
		}
		if got.Path != "/footers/ftr_1/set-default" {
			t.Errorf("Path = %q", got.Path)
		}
	})

	t.Run("rejects an unknown value", func(t *testing.T) {
		stubStdinTerminal(t, false)
		got, _, err := run(t, "some", true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got.Method != "" {
			t.Errorf("request sent for invalid value: %s %s", got.Method, got.Path)
		}
	})
}

func TestRunFootersVariantGet(t *testing.T) {
	t.Run("returns the variant of the footer", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, footerBody)
		v, err := runFootersVariantGet(cfg(t), "ftr_1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Path != "/footers/ftr_1" {
			t.Errorf("Path = %q", got.Path)
		}
		if v.ID != "var_1" || v.ContentRevisionID != "rev_1" || v.PublishedLMX != nil {
			t.Errorf("variant = %+v", v)
		}
	})

	t.Run("returns error when the footer has no variant", func(t *testing.T) {
		serveJSON(t, http.StatusOK, `{"id":"ftr_1","name":"Main","variants":[]}`)
		if _, err := runFootersVariantGet(cfg(t), "ftr_1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunFootersVariantGetMultipleVariants(t *testing.T) {
	serveJSON(t, http.StatusOK, `{"id":"ftr_1","name":"Main","variants":[{"id":"var_1"},{"id":"var_2"}]}`)
	_, err := runFootersVariantGet(cfg(t), "ftr_1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	for _, id := range []string{"var_1", "var_2"} {
		if !strings.Contains(err.Error(), id) {
			t.Errorf("error does not name %s: %v", id, err)
		}
	}
}

func TestRunFootersVariantUpdate(t *testing.T) {
	t.Run("sends lmx and expectedRevisionId", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, footerVariantBody)
		v, err := runFootersVariantUpdate(cfg(t), "ftr_1", "var_1", loops.UpdateFooterVariantRequest{
			ExpectedRevisionID: "rev_1",
			LMX:                "<Paragraph>Draft</Paragraph>",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Method != http.MethodPost || got.Path != "/footers/ftr_1/variants/var_1" {
			t.Errorf("request = %s %s", got.Method, got.Path)
		}
		sent := decodeBody(t, got.Body)
		if sent["expectedRevisionId"] != "rev_1" || sent["lmx"] != "<Paragraph>Draft</Paragraph>" {
			t.Errorf("body = %v", sent)
		}
		if v.DraftLMX != "<Paragraph>Draft</Paragraph>" {
			t.Errorf("DraftLMX = %q", v.DraftLMX)
		}
	})

	t.Run("returns error on 409 revision mismatch", func(t *testing.T) {
		serveJSON(t, http.StatusConflict, `{"success":false,"message":"Revision mismatch"}`)
		_, err := runFootersVariantUpdate(cfg(t), "ftr_1", "var_1", loops.UpdateFooterVariantRequest{ExpectedRevisionID: "rev_stale", LMX: "x"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunFootersVariantPublish(t *testing.T) {
	t.Run("sends expectedRevisionId to the publish path", func(t *testing.T) {
		got := serveJSONCapture(t, http.StatusOK, footerVariantBody)
		_, err := runFootersVariantPublish(cfg(t), "ftr_1", "var_1", loops.PublishFooterVariantRequest{ExpectedRevisionID: "rev_1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Method != http.MethodPost || got.Path != "/footers/ftr_1/variants/var_1/publish" {
			t.Errorf("request = %s %s", got.Method, got.Path)
		}
		if sent := decodeBody(t, got.Body); sent["expectedRevisionId"] != "rev_1" {
			t.Errorf("expectedRevisionId = %v, want rev_1", sent["expectedRevisionId"])
		}
	})

	t.Run("returns error on 422 missing system variables", func(t *testing.T) {
		serveJSON(t, http.StatusUnprocessableEntity, `{"success":false,"message":"Footer must include {system.unsubscribe_link}."}`)
		_, err := runFootersVariantPublish(cfg(t), "ftr_1", "var_1", loops.PublishFooterVariantRequest{ExpectedRevisionID: "rev_1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFooterVariantTargetFromCmd(t *testing.T) {
	newCmd := func(args ...string) *cobra.Command {
		cmd := &cobra.Command{}
		addFooterVariantRevisionFlags(cmd)
		cmd.ParseFlags(args)
		return cmd
	}

	t.Run("--force uses the current revision", func(t *testing.T) {
		serveJSON(t, http.StatusOK, footerBody)
		variantID, revisionID, err := footerVariantTargetFromCmd(newCmd("--force"), cfg(t), "ftr_1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if variantID != "var_1" || revisionID != "rev_1" {
			t.Errorf("target = %q, %q; want var_1, rev_1", variantID, revisionID)
		}
	})

	t.Run("--expected-revision-id is passed through", func(t *testing.T) {
		serveJSON(t, http.StatusOK, footerBody)
		variantID, revisionID, err := footerVariantTargetFromCmd(newCmd("-r", "rev_seen"), cfg(t), "ftr_1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if variantID != "var_1" || revisionID != "rev_seen" {
			t.Errorf("target = %q, %q; want var_1, rev_seen", variantID, revisionID)
		}
	})

	t.Run("returns error when the footer is not found", func(t *testing.T) {
		serveJSON(t, http.StatusNotFound, `{"success":false,"message":"Footer not found."}`)
		if _, _, err := footerVariantTargetFromCmd(newCmd("--force"), cfg(t), "ftr_missing"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPrintFooter(t *testing.T) {
	var f loops.FooterWithVariants
	if err := json.Unmarshal([]byte(footerBody), &f); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	var out strings.Builder
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := printFooter(cmd, &f); err != nil {
		t.Fatalf("printFooter: %v", err)
	}

	got := fieldValueRows(out.String())
	want := map[string]string{
		"name":               "Main",
		"placement":          "below",
		"variantId":          "var_1",
		"contentRevisionId":  "rev_1",
		"publishedAt":        "-",
		"unpublishedChanges": "true",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q\n%s", k, got[k], v, out.String())
		}
	}
	if !strings.Contains(out.String(), "Draft") {
		t.Errorf("draft LMX missing from output:\n%s", out.String())
	}
}
