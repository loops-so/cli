package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
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

		return printSimplifiedWorkflow(cmd, w)
	},
}

func printSimplifiedWorkflow(cmd *cobra.Command, w *loops.SimplifiedWorkflow) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("workflowId", w.ID)
	t.Row("name", w.Name)
	t.Row("description", w.Description)
	t.Row("emoji", w.Emoji)
	t.Row("status", w.Status)
	t.Row("workflowRevisionId", deref(w.WorkflowRevisionID))
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

// createWorkflowNodeTypes is the set of node types that can be created via
// `workflows nodes create`, per the SDK CreateWorkflowNodeType* constants.
var createWorkflowNodeTypes = []string{
	loops.CreateWorkflowNodeTypeAudienceFilter,
	loops.CreateWorkflowNodeTypeBranchNode,
	loops.CreateWorkflowNodeTypeExperimentBranchNode,
	loops.CreateWorkflowNodeTypeTimerAction,
	loops.CreateWorkflowNodeTypeSendEmailAction,
	loops.CreateWorkflowNodeTypeVariantNode,
}

// validateQueuedContactPolicy checks a --queued-contact-policy flag value. An
// empty value is allowed (the API defaults to "fail").
func validateQueuedContactPolicy(v string) error {
	switch v {
	case "", loops.WorkflowQueuedContactPolicyFail, loops.WorkflowQueuedContactPolicyDiscard:
		return nil
	default:
		return fmt.Errorf("--queued-contact-policy must be %q or %q", loops.WorkflowQueuedContactPolicyFail, loops.WorkflowQueuedContactPolicyDiscard)
	}
}

// readExpectedRevisionID reads the optional --expected-revision-id flag,
// returning nil when unset (sent as JSON null for optimistic concurrency).
func readExpectedRevisionID(cmd *cobra.Command) *string {
	if !cmd.Flags().Changed("expected-revision-id") {
		return nil
	}
	v, _ := cmd.Flags().GetString("expected-revision-id")
	return &v
}

// parseUpdateWorkflowNodePayload reads a node-update payload JSON file and
// builds a loops.UpdateWorkflowNodePayload. The file must contain a JSON object
// with a "typeName" discriminator plus the variant fields (the API wire format,
// e.g. {"typeName":"TimerAction","amount":3,"unit":"d"}). We read typeName, then
// unmarshal the same bytes into the matching *Workflow*Payload variant — the SDK
// type has a MarshalJSON but no UnmarshalJSON, so a direct decode cannot populate
// its untagged variant pointers. This mirrors the SDK's own discriminated
// UnmarshalJSON dispatch and round-trips with its MarshalJSON output.
func parseUpdateWorkflowNodePayload(path string) (loops.UpdateWorkflowNodePayload, error) {
	var payload loops.UpdateWorkflowNodePayload

	data, err := os.ReadFile(path)
	if err != nil {
		return payload, fmt.Errorf("read --payload-file: %w", err)
	}

	var head struct {
		TypeName string `json:"typeName"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return payload, fmt.Errorf("parse --payload-file: %w", err)
	}
	if head.TypeName == "" {
		return payload, fmt.Errorf(`--payload-file must include a "typeName" field`)
	}
	payload.TypeName = head.TypeName

	unmarshalInto := func(v any) error {
		if err := json.Unmarshal(data, v); err != nil {
			return fmt.Errorf("parse --payload-file: %w", err)
		}
		return nil
	}

	switch head.TypeName {
	case loops.WorkflowNodeTypeSignupTrigger:
		var v loops.WorkflowSignupTriggerPayload
		if err := unmarshalInto(&v); err != nil {
			return payload, err
		}
		payload.SignupTrigger = &v
	case loops.WorkflowNodeTypeEventTrigger:
		var v loops.WorkflowEventTriggerPayload
		if err := unmarshalInto(&v); err != nil {
			return payload, err
		}
		payload.EventTrigger = &v
	case loops.WorkflowNodeTypeContactPropertyTrigger:
		var v loops.WorkflowContactPropertyTriggerPayload
		if err := unmarshalInto(&v); err != nil {
			return payload, err
		}
		payload.ContactPropertyTrigger = &v
	case loops.WorkflowNodeTypeAddToListTrigger:
		var v loops.WorkflowAddToListTriggerPayload
		if err := unmarshalInto(&v); err != nil {
			return payload, err
		}
		payload.AddToListTrigger = &v
	case loops.WorkflowNodeTypeAudienceFilter:
		var v loops.WorkflowAudienceFilterPayload
		if err := unmarshalInto(&v); err != nil {
			return payload, err
		}
		payload.AudienceFilter = &v
	case loops.WorkflowNodeTypeTimerAction:
		var v loops.WorkflowTimerActionPayload
		if err := unmarshalInto(&v); err != nil {
			return payload, err
		}
		payload.TimerAction = &v
	case loops.WorkflowNodeTypeExperimentBranchNode:
		var v loops.WorkflowExperimentBranchPayload
		if err := unmarshalInto(&v); err != nil {
			return payload, err
		}
		payload.ExperimentBranch = &v
	case loops.WorkflowNodeTypeVariantNode:
		var v loops.WorkflowVariantPayload
		if err := unmarshalInto(&v); err != nil {
			return payload, err
		}
		payload.Variant = &v
	default:
		return payload, fmt.Errorf("unsupported payload typeName %q", head.TypeName)
	}

	return payload, nil
}

func runWorkflowsCreate(cfg *config.Config, req loops.CreateWorkflowRequest) (*loops.SimplifiedWorkflow, error) {
	return newAPIClient(cfg).CreateWorkflow(req)
}

func runWorkflowsUpdate(cfg *config.Config, id string, req loops.UpdateWorkflowPropertiesRequest) (*loops.SimplifiedWorkflow, error) {
	return newAPIClient(cfg).UpdateWorkflow(id, req)
}

func runWorkflowsChangeMailingList(cfg *config.Config, id string, req loops.ChangeWorkflowMailingListRequest) (*loops.ChangeWorkflowMailingListResponse, error) {
	return newAPIClient(cfg).ChangeWorkflowMailingList(id, req)
}

func runWorkflowsNodeCreate(cfg *config.Config, id string, req loops.CreateWorkflowNodeRequest) (*loops.CreateWorkflowNodeResponse, error) {
	return newAPIClient(cfg).CreateWorkflowNode(id, req)
}

func runWorkflowsNodeUpdate(cfg *config.Config, workflowID, nodeID string, req loops.UpdateWorkflowNodeRequest) (*loops.UpdateWorkflowNodeResponse, error) {
	return newAPIClient(cfg).UpdateWorkflowNode(workflowID, nodeID, req)
}

func runWorkflowsNodeAddBranch(cfg *config.Config, workflowID, nodeID string, req loops.AddWorkflowBranchRequest) (*loops.AddWorkflowBranchResponse, error) {
	return newAPIClient(cfg).AddWorkflowBranch(workflowID, nodeID, req)
}

func runWorkflowsNodeDelete(cfg *config.Config, workflowID, nodeID string, recursive bool, req loops.DeleteWorkflowNodeRequest) (*loops.DeleteWorkflowNodeResponse, error) {
	client := newAPIClient(cfg)
	if recursive {
		return client.DeleteWorkflowNodeRecursive(workflowID, nodeID, req)
	}
	return client.DeleteWorkflowNode(workflowID, nodeID, req)
}

func printChangeMailingListResponse(cmd *cobra.Command, r *loops.ChangeWorkflowMailingListResponse) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("status", r.Status)
	t.Row("mailingListId", deref(r.MailingListID))
	t.Row("workflowRevisionId", deref(r.WorkflowRevisionID))
	t.Row("queuedContactCount", formatFloat(r.QueuedContactCount))
	return t.Render()
}

func printDeleteNodeResponse(cmd *cobra.Command, r *loops.DeleteWorkflowNodeResponse) error {
	t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
	t.Row("status", r.Status)
	t.Row("nodeIds", strings.Join(r.NodeIDs, ", "))
	t.Row("workflowRevisionId", deref(r.WorkflowRevisionID))
	t.Row("queuedContactCount", formatFloat(r.QueuedContactCount))
	return t.Render()
}

var workflowsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a workflow",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")

		mailingListID, _, err := readNullableFlag(cmd, "mailing-list-id")
		if err != nil {
			return err
		}

		w, err := runWorkflowsCreate(cfg, loops.CreateWorkflowRequest{
			Name:          name,
			Description:   description,
			MailingListID: mailingListID,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), w)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created. (id: %s)\n\n", w.ID)
		return printSimplifiedWorkflow(cmd, w)
	},
}

var workflowsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a workflow's name and/or description",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")

		w, err := runWorkflowsUpdate(cfg, args[0], loops.UpdateWorkflowPropertiesRequest{
			ExpectedRevisionID: readExpectedRevisionID(cmd),
			Name:               name,
			Description:        description,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), w)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated. (id: %s)\n\n", w.ID)
		return printSimplifiedWorkflow(cmd, w)
	},
}

var workflowsChangeMailingListCmd = &cobra.Command{
	Use:   "change-mailing-list <id>",
	Short: "Change a workflow's mailing list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		policy, _ := cmd.Flags().GetString("queued-contact-policy")
		if err := validateQueuedContactPolicy(policy); err != nil {
			return err
		}

		mailingListID, _, err := readNullableFlag(cmd, "mailing-list-id")
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		r, err := runWorkflowsChangeMailingList(cfg, args[0], loops.ChangeWorkflowMailingListRequest{
			ExpectedRevisionID:  readExpectedRevisionID(cmd),
			MailingListID:       mailingListID,
			DryRun:              dryRun,
			QueuedContactPolicy: policy,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), r)
		}

		return printChangeMailingListResponse(cmd, r)
	},
}

// parseCreateWorkflowNodeFlags reads and validates the `workflows nodes create`
// flags and builds a CreateWorkflowNodeRequest. Placement depends on insert
// mode: "between" needs --from-node-id and --to-node-id; "before" inserts
// before --before-node-id; "after" inserts after --from-node-id (valid only
// when that node has exactly one outgoing connection). "before" is sent as
// toNodeId because the API's beforeNodeId field is deprecated.
func parseCreateWorkflowNodeFlags(cmd *cobra.Command) (loops.CreateWorkflowNodeRequest, error) {
	nodeType, _ := cmd.Flags().GetString("node-type")
	insertMode, _ := cmd.Flags().GetString("insert-mode")
	fromNodeID, _ := cmd.Flags().GetString("from-node-id")
	toNodeID, _ := cmd.Flags().GetString("to-node-id")
	beforeNodeID, _ := cmd.Flags().GetString("before-node-id")

	if !slices.Contains(createWorkflowNodeTypes, nodeType) {
		return loops.CreateWorkflowNodeRequest{}, fmt.Errorf("--node-type must be one of: %s", strings.Join(createWorkflowNodeTypes, ", "))
	}

	req := loops.CreateWorkflowNodeRequest{
		ExpectedRevisionID: readExpectedRevisionID(cmd),
		InsertMode:         insertMode,
		NodeTypeName:       nodeType,
	}

	switch insertMode {
	case loops.WorkflowInsertModeBetween:
		if fromNodeID == "" || toNodeID == "" {
			return loops.CreateWorkflowNodeRequest{}, fmt.Errorf("--insert-mode between requires --from-node-id and --to-node-id")
		}
		req.FromNodeID = fromNodeID
		req.ToNodeID = toNodeID
	case loops.WorkflowInsertModeBefore:
		if beforeNodeID == "" {
			return loops.CreateWorkflowNodeRequest{}, fmt.Errorf("--insert-mode before requires --before-node-id")
		}
		req.ToNodeID = beforeNodeID
	case loops.WorkflowInsertModeAfter:
		if fromNodeID == "" {
			return loops.CreateWorkflowNodeRequest{}, fmt.Errorf("--insert-mode after requires --from-node-id")
		}
		req.FromNodeID = fromNodeID
	default:
		return loops.CreateWorkflowNodeRequest{}, fmt.Errorf("--insert-mode must be %q, %q, or %q", loops.WorkflowInsertModeBetween, loops.WorkflowInsertModeBefore, loops.WorkflowInsertModeAfter)
	}

	return req, nil
}

var workflowsNodesCreateCmd = &cobra.Command{
	Use:   "create <workflow-id>",
	Short: "Create a workflow node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		req, err := parseCreateWorkflowNodeFlags(cmd)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		resp, err := runWorkflowsNodeCreate(cfg, args[0], req)
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), resp)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created node. (id: %s, type: %s, revision: %s)\n\n", mutationNodeID(&resp.Node.WorkflowMutationNode), resp.Node.TypeName, resp.Node.WorkflowRevisionID)
		return printSimplifiedWorkflow(cmd, &resp.Workflow)
	},
}

var workflowsNodesUpdateCmd = &cobra.Command{
	Use:   "update <workflow-id> <node-id>",
	Short: "Update a workflow node",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("payload-file")
		payload, err := parseUpdateWorkflowNodePayload(path)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		node, err := runWorkflowsNodeUpdate(cfg, args[0], args[1], loops.UpdateWorkflowNodeRequest{
			ExpectedRevisionID: readExpectedRevisionID(cmd),
			Payload:            payload,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), node)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated node. (revision: %s)\n\n", node.WorkflowRevisionID)
		t := newStyledTable(cmd.OutOrStdout(), "FIELD", "VALUE")
		t.Row("typeName", node.TypeName)
		t.Row("nodeId", mutationNodeID(&node.WorkflowMutationNode))
		t.Row("workflowRevisionId", node.WorkflowRevisionID)
		return t.Render()
	},
}

var workflowsNodesAddBranchCmd = &cobra.Command{
	Use:   "add-branch <workflow-id> <node-id>",
	Short: "Add a branch to a branch node",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		resp, err := runWorkflowsNodeAddBranch(cfg, args[0], args[1], loops.AddWorkflowBranchRequest{
			ExpectedRevisionID: readExpectedRevisionID(cmd),
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), resp)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Added branch. (revision: %s)\n\n", resp.Node.WorkflowRevisionID)
		return printSimplifiedWorkflow(cmd, &resp.Workflow)
	},
}

var workflowsNodesDeleteCmd = &cobra.Command{
	Use:   "delete <workflow-id> <node-id>",
	Short: "Delete a workflow node",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		policy, _ := cmd.Flags().GetString("queued-contact-policy")
		if err := validateQueuedContactPolicy(policy); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		recursive, _ := cmd.Flags().GetBool("recursive")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		r, err := runWorkflowsNodeDelete(cfg, args[0], args[1], recursive, loops.DeleteWorkflowNodeRequest{
			ExpectedRevisionID:  readExpectedRevisionID(cmd),
			DryRun:              dryRun,
			QueuedContactPolicy: policy,
		})
		if err != nil {
			return err
		}

		if isJSONOutput() {
			return printJSON(cmd.OutOrStdout(), r)
		}

		return printDeleteNodeResponse(cmd, r)
	},
}

// mutationNodeID returns the ID of the active variant of a WorkflowMutationNode.
func mutationNodeID(n *loops.WorkflowMutationNode) string {
	switch n.TypeName {
	case loops.WorkflowNodeTypeSignupTrigger:
		if n.SignupTrigger != nil {
			return n.SignupTrigger.ID
		}
	case loops.WorkflowNodeTypeEventTrigger:
		if n.EventTrigger != nil {
			return n.EventTrigger.ID
		}
	case loops.WorkflowNodeTypeContactPropertyTrigger:
		if n.ContactPropertyTrigger != nil {
			return n.ContactPropertyTrigger.ID
		}
	case loops.WorkflowNodeTypeAddToListTrigger:
		if n.AddToListTrigger != nil {
			return n.AddToListTrigger.ID
		}
	case loops.WorkflowNodeTypeBlankTrigger:
		if n.BlankTrigger != nil {
			return n.BlankTrigger.ID
		}
	case loops.WorkflowNodeTypeAudienceFilter:
		if n.AudienceFilter != nil {
			return n.AudienceFilter.ID
		}
	case loops.WorkflowNodeTypeTimerAction:
		if n.TimerAction != nil {
			return n.TimerAction.ID
		}
	case loops.WorkflowNodeTypeSendEmailAction:
		if n.SendEmailAction != nil {
			return n.SendEmailAction.ID
		}
	case loops.WorkflowNodeTypeExitAction:
		if n.ExitAction != nil {
			return n.ExitAction.ID
		}
	case loops.WorkflowNodeTypeBranchNode:
		if n.BranchNode != nil {
			return n.BranchNode.ID
		}
	case loops.WorkflowNodeTypeExperimentBranchNode:
		if n.ExperimentBranchNode != nil {
			return n.ExperimentBranchNode.ID
		}
	case loops.WorkflowNodeTypeVariantNode:
		if n.VariantNode != nil {
			return n.VariantNode.ID
		}
	}
	return ""
}

func init() {
	addPaginationFlags(workflowsListCmd)
	addPickFlag(workflowsListCmd)
	workflowsCmd.AddCommand(workflowsListCmd)
	workflowsCmd.AddCommand(workflowsGetCmd)

	workflowsCreateCmd.Flags().StringP("name", "n", "", "Workflow name")
	workflowsCreateCmd.Flags().StringP("description", "d", "", "Workflow description")
	workflowsCreateCmd.Flags().String("mailing-list-id", "", `Mailing list ID. Pass "null" to clear.`)
	workflowsCreateCmd.MarkFlagRequired("name")
	workflowsCmd.AddCommand(workflowsCreateCmd)

	workflowsUpdateCmd.Flags().StringP("name", "n", "", "Workflow name")
	workflowsUpdateCmd.Flags().StringP("description", "d", "", "Workflow description")
	workflowsUpdateCmd.Flags().String("expected-revision-id", "", "Expected workflow revision ID (optimistic concurrency)")
	workflowsUpdateCmd.MarkFlagsOneRequired("name", "description")
	workflowsCmd.AddCommand(workflowsUpdateCmd)

	workflowsChangeMailingListCmd.Flags().String("mailing-list-id", "", `Mailing list ID. Pass "null" to clear.`)
	workflowsChangeMailingListCmd.Flags().String("expected-revision-id", "", "Expected workflow revision ID (optimistic concurrency)")
	workflowsChangeMailingListCmd.Flags().Bool("dry-run", false, "Report queued-contact impact without applying the change")
	workflowsChangeMailingListCmd.Flags().String("queued-contact-policy", "", "How to treat queued contacts: fail or discard")
	workflowsChangeMailingListCmd.MarkFlagRequired("mailing-list-id")
	workflowsCmd.AddCommand(workflowsChangeMailingListCmd)

	workflowsNodesCmd.AddCommand(workflowsNodesGetCmd)

	workflowsNodesCreateCmd.Flags().String("node-type", "", fmt.Sprintf("Node type: %s", strings.Join(createWorkflowNodeTypes, ", ")))
	workflowsNodesCreateCmd.Flags().String("insert-mode", "", "Insert mode: between, before, or after")
	workflowsNodesCreateCmd.Flags().String("from-node-id", "", "Source node ID (insert-mode between or after)")
	workflowsNodesCreateCmd.Flags().String("to-node-id", "", "Target node ID (insert-mode between)")
	workflowsNodesCreateCmd.Flags().String("before-node-id", "", "Node ID to insert before (insert-mode before)")
	workflowsNodesCreateCmd.Flags().String("expected-revision-id", "", "Expected workflow revision ID (optimistic concurrency)")
	workflowsNodesCreateCmd.MarkFlagRequired("node-type")
	workflowsNodesCreateCmd.MarkFlagRequired("insert-mode")
	workflowsNodesCmd.AddCommand(workflowsNodesCreateCmd)

	workflowsNodesUpdateCmd.Flags().String("payload-file", "", "Path to a JSON file with the node payload (must include a typeName field)")
	workflowsNodesUpdateCmd.Flags().String("expected-revision-id", "", "Expected workflow revision ID (optimistic concurrency)")
	workflowsNodesUpdateCmd.MarkFlagRequired("payload-file")
	workflowsNodesCmd.AddCommand(workflowsNodesUpdateCmd)

	workflowsNodesAddBranchCmd.Flags().String("expected-revision-id", "", "Expected workflow revision ID (optimistic concurrency)")
	workflowsNodesCmd.AddCommand(workflowsNodesAddBranchCmd)

	workflowsNodesDeleteCmd.Flags().Bool("recursive", false, "Also delete downstream nodes")
	workflowsNodesDeleteCmd.Flags().String("expected-revision-id", "", "Expected workflow revision ID (optimistic concurrency)")
	workflowsNodesDeleteCmd.Flags().Bool("dry-run", false, "Report queued-contact impact without applying the deletion")
	workflowsNodesDeleteCmd.Flags().String("queued-contact-policy", "", "How to treat queued contacts: fail or discard")
	workflowsNodesCmd.AddCommand(workflowsNodesDeleteCmd)

	workflowsCmd.AddCommand(workflowsNodesCmd)
	rootCmd.AddCommand(workflowsCmd)
}
