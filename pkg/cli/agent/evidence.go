package agent

// Features implemented: agent-coordination

import (
	"strings"

	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state"
)

// parseEvidenceFlags parses repeatable --evidence kind:description:reference
// values into typed state.EvidenceRef entries — no anonymous maps make it
// into the domain call. description/reference may themselves be empty
// (e.g. "counterexample::run-b-claim-1"); only kind is required per entry.
func parseEvidenceFlags(values []string) ([]state.EvidenceRef, error) {
	if len(values) == 0 {
		return nil, nil
	}
	refs := make([]state.EvidenceRef, 0, len(values))
	for _, raw := range values {
		parts := strings.SplitN(raw, ":", 3)
		kind := strings.TrimSpace(parts[0])
		if kind == "" {
			return nil, exitcode.InvalidArgsErrorf("--evidence %q: kind is required (expected kind:description:reference)", raw)
		}
		var description, reference string
		if len(parts) > 1 {
			description = parts[1]
		}
		if len(parts) > 2 {
			reference = parts[2]
		}
		refs = append(refs, state.EvidenceRef{Kind: kind, Description: description, Reference: reference})
	}
	return refs, nil
}
