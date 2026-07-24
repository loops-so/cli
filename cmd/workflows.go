package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/loops-so/cli/internal/config"
	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

func runWorkflowsList(cfg *config.Config, params loops.PaginationParams) ([]loops.WorkflowSummary, error) {
	client := newAPIClient(cfg)
	if params.Cursor != "" {
		workflows, _, err := client.ListWorkflows(params)
		return workflows, err
	}
	return loops.Paginate(func(cursor string) ([]loops.WorkflowSummary, *loops.Pagination, error) {
		return client.ListWorkflows(loops.PaginationParams{
			PerPage: params.PerPage,
			Cursor:  cursor,
		})
	})
}

func runWorkflowsGet(cfg *config.Config, id string) (*loops.SimplifiedWorkflow, error) {
	return newAPIClient(cfg).GetWorkflow(id)
}

func runWorkflowsNodeGet(cfg *config.Config, workflowID, nodeID string) (*loops.WorkflowNodeWithRevision, error) {
	return newAPIClient(cfg).GetWorkflowNode(workflowID, nodeID)
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
				workflows = []loops.WorkflowSummary{}
			}
			return printJSON(cmd.OutOrStdout(), workflows)
		}

		if len(workflows) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No workflows found.")
			return nil
		}

		headers := []string{"ID", "NAME", "CREATED", "UPDATED"}
		rows := make([][]string, 0, len(workflows))
		for _, w := range workflows {
			rows = append(rows, []string{w.ID, w.Name, w.CreatedAt, w.UpdatedAt})
		}

		if isPicking(cmd) {
			return runPicker(headers, rows, []pickBinding{
				copyColumnBinding("enter", "copy id", "workflow ID", rows, 0, cmd.OutOrStdout()),
			})
		}

		t := newStyledTable(cmd.OutOrStdout(), headers...)
		for _, r := range rows {
			t.Row(r...)
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

		w, err := runWorkflowsGet(cfg, args[0])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), w)
		}

		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("workflowId", w.ID)
		t.Row("name", w.Name)
		t.Row("description", w.Description)
		t.Row("emoji", w.Emoji)
		t.Row("mailingListId", deref(w.MailingListID))
		t.Row("rootNodeId", deref(w.RootNodeID))
		if err := t.Render(); err != nil {
			return err
		}

		if len(w.Nodes) == 0 {
			return nil
		}

		fmt.Fprintln(cmd.OutOrStdout())
		nt := newStyledTable(cmd.OutOrStdout(), "NODE ID", "TYPE", "NEXT IDS")
		ids := make([]string, 0, len(w.Nodes))
		for id := range w.Nodes {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			n := w.Nodes[id]
			nt.Row(id, n.TypeName, strings.Join(simplifiedNodeNextIDs(n), ", "))
		}
		return nt.Render()
	},
}

var workflowsNodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Manage workflow nodes",
}

var workflowsNodesGetCmd = &cobra.Command{
	Use:   "get <workflow-id> <node-id>",
	Short: "Get a workflow node",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		n, err := runWorkflowsNodeGet(cfg, args[0], args[1])
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), n)
		}

		return printWorkflowNode(cmd, &n.WorkflowNode)
	},
}

func simplifiedNodeNextIDs(n loops.SimplifiedWorkflowNode) []string {
	switch n.TypeName {
	case loops.WorkflowNodeTypeSignupTrigger:
		if n.SignupTrigger != nil {
			return n.SignupTrigger.NextNodeIDs
		}
	case loops.WorkflowNodeTypeEventTrigger:
		if n.EventTrigger != nil {
			return n.EventTrigger.NextNodeIDs
		}
	case loops.WorkflowNodeTypeContactPropertyTrigger:
		if n.ContactPropertyTrigger != nil {
			return n.ContactPropertyTrigger.NextNodeIDs
		}
	case loops.WorkflowNodeTypeAddToListTrigger:
		if n.AddToListTrigger != nil {
			return n.AddToListTrigger.NextNodeIDs
		}
	case loops.WorkflowNodeTypeBlankTrigger:
		if n.BlankTrigger != nil {
			return n.BlankTrigger.NextNodeIDs
		}
	case loops.WorkflowNodeTypeAudienceFilter:
		if n.AudienceFilter != nil {
			return n.AudienceFilter.NextNodeIDs
		}
	case loops.WorkflowNodeTypeTimerAction:
		if n.TimerAction != nil {
			return n.TimerAction.NextNodeIDs
		}
	case loops.WorkflowNodeTypeSendEmailAction:
		if n.SendEmailAction != nil {
			return n.SendEmailAction.NextNodeIDs
		}
	case loops.WorkflowNodeTypeExitAction:
		if n.ExitAction != nil {
			return n.ExitAction.NextNodeIDs
		}
	case loops.WorkflowNodeTypeBranchNode:
		if n.BranchNode != nil {
			return n.BranchNode.NextNodeIDs
		}
	case loops.WorkflowNodeTypeExperimentBranchNode:
		if n.ExperimentBranchNode != nil {
			return n.ExperimentBranchNode.NextNodeIDs
		}
	case loops.WorkflowNodeTypeVariantNode:
		if n.VariantNode != nil {
			return n.VariantNode.NextNodeIDs
		}
	}
	return nil
}

func printWorkflowNode(cmd *cobra.Command, n *loops.WorkflowNode) error {
	rows := workflowNodeRows(n)
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	for _, r := range rows {
		t.Row(r[0], r[1])
	}
	return t.Render()
}

func workflowNodeRows(n *loops.WorkflowNode) [][2]string {
	rows := [][2]string{{"typeName", n.TypeName}}
	add := func(field, value string) { rows = append(rows, [2]string{field, value}) }
	addCommon := func(id, workflowID string, nextIDs []string) {
		add("nodeId", id)
		add("workflowId", workflowID)
		add("nextNodeIds", strings.Join(nextIDs, ", "))
	}
	switch n.TypeName {
	case loops.WorkflowNodeTypeSignupTrigger:
		if v := n.SignupTrigger; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
		}
	case loops.WorkflowNodeTypeEventTrigger:
		if v := n.EventTrigger; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
			add("eventName", deref(v.EventName))
			add("reEligible", strconv.FormatBool(v.ReEligible))
			add("eventProperties", fmt.Sprintf("%d properties", len(v.EventProperties)))
		}
	case loops.WorkflowNodeTypeContactPropertyTrigger:
		if v := n.ContactPropertyTrigger; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
			add("reEligible", strconv.FormatBool(v.ReEligible))
			if v.ContactPropertyQuery != nil {
				add("contactPropertyQuery", fmt.Sprintf("key=%s (see -o json)", v.ContactPropertyQuery.Key))
			} else {
				add("contactPropertyQuery", "")
			}
		}
	case loops.WorkflowNodeTypeAddToListTrigger:
		if v := n.AddToListTrigger; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
			add("mailingListId", deref(v.MailingListID))
			add("reEligible", strconv.FormatBool(v.ReEligible))
		}
	case loops.WorkflowNodeTypeBlankTrigger:
		if v := n.BlankTrigger; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
		}
	case loops.WorkflowNodeTypeAudienceFilter:
		if v := n.AudienceFilter; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
			add("audienceSegmentId", v.AudienceSegmentID)
			add("audienceFilter", formatAudienceFilter(v.AudienceFilter))
		}
	case loops.WorkflowNodeTypeTimerAction:
		if v := n.TimerAction; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
			add("amount", formatFloat(v.Amount))
			add("unit", string(v.Unit))
		}
	case loops.WorkflowNodeTypeSendEmailAction:
		if v := n.SendEmailAction; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
			add("emailMessageId", v.EmailMessageID)
			add("subject", v.Subject)
		}
	case loops.WorkflowNodeTypeExitAction:
		if v := n.ExitAction; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
		}
	case loops.WorkflowNodeTypeBranchNode:
		if v := n.BranchNode; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
			add("evalStrategy", v.EvalStrategy)
		}
	case loops.WorkflowNodeTypeExperimentBranchNode:
		if v := n.ExperimentBranchNode; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
			add("samplingRate", formatFloat(v.SamplingRate))
			add("url", v.URL)
			add("experimentId", v.ExperimentID)
			add("experimentType", string(v.ExperimentType))
		}
	case loops.WorkflowNodeTypeVariantNode:
		if v := n.VariantNode; v != nil {
			addCommon(v.ID, v.WorkflowID, v.NextNodeIDs)
			add("variantId", v.VariantID)
			if v.IsControl != nil {
				add("isControl", strconv.FormatBool(*v.IsControl))
			}
		}
	}
	return rows
}

func init() {
	addPaginationFlags(workflowsListCmd)
	addPickFlag(workflowsListCmd)
	workflowsCmd.AddCommand(workflowsListCmd)
	workflowsCmd.AddCommand(workflowsGetCmd)
	workflowsNodesCmd.AddCommand(workflowsNodesGetCmd)
	workflowsCmd.AddCommand(workflowsNodesCmd)
	rootCmd.AddCommand(workflowsCmd)
}
