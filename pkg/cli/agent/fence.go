package agent

// Features implemented: agent-coordination

import (
	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state"
)

// addFenceFlags registers the --fence-epoch/--fence-token flags shared by
// every command that proves current authority over an existing claim/lease
// (renew, release, handoff). A caller learns these from the claim/lease
// value a prior claim/renew/handoff call returned — it is the CLI-visible
// form of state.LeaseFence.
func addFenceFlags(cmd *cobra.Command) {
	cmd.Flags().Int64("fence-epoch", 0, "authority epoch from the claim's fence (required)")
	cmd.Flags().String("fence-token", "", "opaque fence token from the claim's fence (required)")
}

func readFenceFlags(cmd *cobra.Command) (state.LeaseFence, error) {
	epoch, _ := cmd.Flags().GetInt64("fence-epoch")
	token, _ := cmd.Flags().GetString("fence-token")
	fence := state.LeaseFence{Epoch: epoch, Token: token}
	if fence.IsZero() {
		return state.LeaseFence{}, exitcode.InvalidArgsError("--fence-epoch and --fence-token are required")
	}
	return fence, nil
}
