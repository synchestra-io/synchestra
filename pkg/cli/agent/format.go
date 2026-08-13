package agent

// Features implemented: cli
// Features depended on:  agent-coordination

import (
	"encoding/json"
	"io"

	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state"
	"gopkg.in/yaml.v3"
)

// claimOutput is the serialisable view of a state.WorktreeClaim for CLI
// output, following pkg/cli/task/format.go's taskOutput convention (a
// dedicated wire type rather than serializing the domain struct directly,
// so field names/omission are a deliberate CLI contract).
type claimOutput struct {
	ID           string   `json:"id" yaml:"id"`
	ProjectID    string   `json:"project_id" yaml:"project_id"`
	RepositoryID string   `json:"repository_id" yaml:"repository_id"`
	RunID        string   `json:"run_id" yaml:"run_id"`
	WorktreePath string   `json:"worktree_path,omitempty" yaml:"worktree_path,omitempty"`
	Branch       string   `json:"branch" yaml:"branch"`
	TargetRef    string   `json:"target_ref,omitempty" yaml:"target_ref,omitempty"`
	BaseSHA      string   `json:"base_sha,omitempty" yaml:"base_sha,omitempty"`
	HeadSHA      string   `json:"head_sha,omitempty" yaml:"head_sha,omitempty"`
	ScopeAreas   []string `json:"scope_areas,omitempty" yaml:"scope_areas,omitempty"`
	FenceEpoch   int64    `json:"fence_epoch" yaml:"fence_epoch"`
	FenceToken   string   `json:"fence_token" yaml:"fence_token"`
	ClaimedAt    string   `json:"claimed_at" yaml:"claimed_at"`
	RenewedAt    string   `json:"renewed_at" yaml:"renewed_at"`
	ReleasedAt   string   `json:"released_at,omitempty" yaml:"released_at,omitempty"`
}

func toClaimOutput(c state.WorktreeClaim) claimOutput {
	o := claimOutput{
		ID: c.ID, ProjectID: c.ProjectID, RepositoryID: c.RepositoryID, RunID: c.RunID,
		WorktreePath: c.WorktreePath, Branch: c.Branch, TargetRef: c.TargetRef,
		BaseSHA: c.BaseSHA, HeadSHA: c.HeadSHA, ScopeAreas: c.ScopeAreas,
		FenceEpoch: c.Fence.Epoch, FenceToken: c.Fence.Token,
		ClaimedAt: c.ClaimedAt.Format(timeFormat), RenewedAt: c.RenewedAt.Format(timeFormat),
	}
	if c.ReleasedAt != nil {
		o.ReleasedAt = c.ReleasedAt.Format(timeFormat)
	}
	return o
}

const timeFormat = "2006-01-02T15:04:05Z"

func writeClaim(w io.Writer, format string, claim state.WorktreeClaim) error {
	return writeValue(w, format, toClaimOutput(claim))
}

func writeClaimList(w io.Writer, format string, claims []state.WorktreeClaim) error {
	outputs := make([]claimOutput, len(claims))
	for i, c := range claims {
		outputs[i] = toClaimOutput(c)
	}
	return writeValue(w, format, outputs)
}

// runOutput is the serialisable view of a state.Run for CLI output.
type runOutput struct {
	ID              string `json:"id" yaml:"id"`
	EffortID        string `json:"effort_id" yaml:"effort_id"`
	AgentFamily     string `json:"agent_family" yaml:"agent_family"`
	Model           string `json:"model,omitempty" yaml:"model,omitempty"`
	ModelProvenance string `json:"model_provenance,omitempty" yaml:"model_provenance,omitempty"`
	Role            string `json:"role" yaml:"role"`
	ParentRunID     string `json:"parent_run_id,omitempty" yaml:"parent_run_id,omitempty"`
	Status          string `json:"status" yaml:"status"`
	StartedAt       string `json:"started_at" yaml:"started_at"`
	EndedAt         string `json:"ended_at,omitempty" yaml:"ended_at,omitempty"`
	TerminalReason  string `json:"terminal_reason,omitempty" yaml:"terminal_reason,omitempty"`
}

func toRunOutput(r state.Run) runOutput {
	o := runOutput{
		ID: r.ID, EffortID: r.EffortID, AgentFamily: string(r.AgentFamily),
		ModelProvenance: string(r.ModelProvenance), Role: string(r.Role), ParentRunID: r.ParentRunID,
		Status: string(r.Status), StartedAt: r.StartedAt.Format(timeFormat), TerminalReason: r.TerminalReason,
	}
	if r.Model != nil {
		o.Model = *r.Model
	}
	if r.EndedAt != nil {
		o.EndedAt = r.EndedAt.Format(timeFormat)
	}
	return o
}

func writeRun(w io.Writer, format string, run state.Run) error {
	return writeValue(w, format, toRunOutput(run))
}

// messageOutput is the serialisable view of a state.Message for CLI output.
type messageOutput struct {
	ID              string              `json:"id" yaml:"id"`
	EffortID        string              `json:"effort_id,omitempty" yaml:"effort_id,omitempty"`
	ThreadID        string              `json:"thread_id" yaml:"thread_id"`
	SenderRunID     string              `json:"sender_run_id" yaml:"sender_run_id"`
	RecipientRunIDs []string            `json:"recipient_run_ids,omitempty" yaml:"recipient_run_ids,omitempty"`
	CorrelationID   string              `json:"correlation_id,omitempty" yaml:"correlation_id,omitempty"`
	RepositoryID    string              `json:"repository_id,omitempty" yaml:"repository_id,omitempty"`
	ClaimID         string              `json:"claim_id,omitempty" yaml:"claim_id,omitempty"`
	Kind            string              `json:"kind,omitempty" yaml:"kind,omitempty"`
	Body            string              `json:"body,omitempty" yaml:"body,omitempty"`
	Evidence        []evidenceRefOutput `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	SentAt          string              `json:"sent_at" yaml:"sent_at"`
}

type evidenceRefOutput struct {
	Kind        string `json:"kind,omitempty" yaml:"kind,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Reference   string `json:"reference,omitempty" yaml:"reference,omitempty"`
}

func toMessageOutput(m state.Message) messageOutput {
	o := messageOutput{
		ID: m.ID, EffortID: m.EffortID, ThreadID: m.ThreadID, SenderRunID: m.SenderRunID,
		RecipientRunIDs: m.RecipientRunIDs, CorrelationID: m.CorrelationID, RepositoryID: m.RepositoryID,
		ClaimID: m.ClaimID, Kind: string(m.Kind), Body: m.Body, SentAt: m.SentAt.Format(timeFormat),
	}
	for _, e := range m.Evidence {
		o.Evidence = append(o.Evidence, evidenceRefOutput{Kind: e.Kind, Description: e.Description, Reference: e.Reference})
	}
	return o
}

func writeMessage(w io.Writer, format string, msg state.Message) error {
	return writeValue(w, format, toMessageOutput(msg))
}

func writeMessageList(w io.Writer, format string, messages []state.Message) error {
	outputs := make([]messageOutput, len(messages))
	for i, m := range messages {
		outputs[i] = toMessageOutput(m)
	}
	return writeValue(w, format, outputs)
}

// writeValue renders v as YAML (the CLI's agent-first default, per
// spec/features/cli/feature/new/README.md's "Default format is YAML") or
// JSON with --format json; any other value is a validation error.
func writeValue(w io.Writer, format string, v any) error {
	switch format {
	case "", "yaml":
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		if err := enc.Encode(v); err != nil {
			return err
		}
		return enc.Close()
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	default:
		return exitcode.InvalidArgsErrorf("invalid --format value %q: must be yaml or json", format)
	}
}
