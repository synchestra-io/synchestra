package state

// Features implemented: cli/state
// Features depended on:  project-definition

import (
	"github.com/synchestra-io/synchestra/pkg/cli/resolve"
)

// TODO: Remove once pull/push/sync commands call resolve.StateRepoPath directly.
var _ = resolve.StateRepoPath
