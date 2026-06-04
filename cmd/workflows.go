package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

type WorkflowListItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type WorkflowEmailMessage struct {
	ID                string   `json:"id"`
	EmailMessageID    string   `json:"emailMessageId"`
	NodeID            *string  `json:"nodeId"`
	NodeIDs           []string `json:"nodeIds"`
	Subject           string   `json:"subject"`
	PreviewText       string   `json:"previewText"`
	FromName          string   `json:"fromName"`
	FromEmail         string   `json:"fromEmail"`
	ReplyToEmail      string   `json:"replyToEmail"`
	ContentRevisionID *string  `json:"contentRevisionId"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
}

type WorkflowNode struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	EmailMessageID string `json:"emailMessageId"`
}

type WorkflowDetail struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	CreatedAt     string                 `json:"createdAt"`
	UpdatedAt     string                 `json:"updatedAt"`
	EmailMessages []WorkflowEmailMessage `json:"emailMessages"`
	Nodes         []WorkflowNode         `json:"nodes"`
}

type workflowAPIError struct {
	statusCode int
	message    string
}

func (e workflowAPIError) Error() string { return e.message }

func workflowURL(cfg *config.Config, path string, params loops.PaginationParams) string {
	q := url.Values{}
	if params.PerPage != "" {
		q.Set("perPage", params.PerPage)
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}

	u := strings.TrimRight(cfg.EndpointURL, "/") + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

func workflowErrorFromResponse(resp *http.Response) error {
	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
		if body.Error != "" {
			return workflowAPIError{statusCode: resp.StatusCode, message: body.Error}
		}
		if body.Message != "" {
			return workflowAPIError{statusCode: resp.StatusCode, message: body.Message}
		}
	}
	return workflowAPIError{
		statusCode: resp.StatusCode,
		message:    fmt.Sprintf("unexpected status: %d", resp.StatusCode),
	}
}

func runWorkflowsListPage(cfg *config.Config, params loops.PaginationParams) ([]WorkflowListItem, *loops.Pagination, error) {
	req, err := http.NewRequest(http.MethodGet, workflowURL(cfg, "/workflows", params), nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("User-Agent", "loops-cli/"+version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, workflowErrorFromResponse(resp)
	}

	var result struct {
		Pagination loops.Pagination   `json:"pagination"`
		Data       []WorkflowListItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return result.Data, &result.Pagination, nil
}

func runWorkflowsGet(cfg *config.Config, id string) (*WorkflowDetail, error) {
	req, err := http.NewRequest(http.MethodGet, workflowURL(cfg, "/workflows/"+url.PathEscape(id), loops.PaginationParams{}), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("User-Agent", "loops-cli/"+version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, workflowErrorFromResponse(resp)
	}

	var workflow WorkflowDetail
	if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &workflow, nil
}

func runWorkflowsList(cfg *config.Config, params loops.PaginationParams) ([]WorkflowListItem, error) {
	if params.Cursor != "" {
		workflows, _, err := runWorkflowsListPage(cfg, params)
		return workflows, err
	}
	return loops.Paginate(func(cursor string) ([]WorkflowListItem, *loops.Pagination, error) {
		return runWorkflowsListPage(cfg, loops.PaginationParams{
			PerPage: params.PerPage,
			Cursor:  cursor,
		})
	})
}

var workflowsCmd = &cobra.Command{
	Use:   "workflows",
	Short: "Manage workflows",
}

var workflowsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflows",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePickFlags(cmd); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		workflows, err := runWorkflowsList(cfg, paginationParams(cmd))
		if err != nil {
			return err
		}

		if isJSONOutput() {
			if workflows == nil {
				workflows = []WorkflowListItem{}
			}
			return printJSON(cmd.OutOrStdout(), workflows)
		}

		if len(workflows) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No workflows found.")
			return nil
		}

		headers := []string{"ID", "NAME", "UPDATED"}
		rows := make([][]string, 0, len(workflows))
		for _, workflow := range workflows {
			rows = append(rows, []string{
				workflow.ID,
				workflow.Name,
				workflow.UpdatedAt,
			})
		}

		if isPicking(cmd) {
			return runPicker(headers, rows, []pickBinding{
				copyColumnBinding("enter", "copy id", "workflow ID", rows, 0, cmd.OutOrStdout()),
			})
		}

		t := newStyledTable(cmd.OutOrStdout(), headers...)
		for _, row := range rows {
			t.Row(row...)
		}
		return t.Render()
	},
}

var workflowsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		workflow, err := runWorkflowsGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), workflow)
		}

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("workflowId", workflow.ID)
		t.Row("name", workflow.Name)
		t.Row("status", workflow.Status)
		t.Row("createdAt", workflow.CreatedAt)
		t.Row("updatedAt", workflow.UpdatedAt)
		if err := t.Render(); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout())
		if len(workflow.EmailMessages) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No workflow email messages found.")
			return nil
		}

		messageTable := newStyledTable(cmd.OutOrStdout(), "MESSAGE ID", "NODE ID", "SUBJECT", "REVISION", "UPDATED")
		for _, emailMessage := range workflow.EmailMessages {
			messageTable.Row(
				emailMessage.EmailMessageID,
				deref(emailMessage.NodeID),
				emailMessage.Subject,
				deref(emailMessage.ContentRevisionID),
				emailMessage.UpdatedAt,
			)
		}
		return messageTable.Render()
	},
}

func init() {
	addPaginationFlags(workflowsListCmd)
	addPickFlag(workflowsListCmd)
	workflowsCmd.AddCommand(workflowsListCmd)
	workflowsCmd.AddCommand(workflowsGetCmd)
	rootCmd.AddCommand(workflowsCmd)
}
