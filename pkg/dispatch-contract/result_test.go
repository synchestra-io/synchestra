package dispatchcontract_test

// Features implemented: dispatch, dispatch/scheduler, dispatch/worker

import (
	"testing"
	"time"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

func TestBranchResultRequiresPublicationTimestamp(t *testing.T) {
	t.Parallel()

	result := dispatchcontract.BranchResult{
		RepositoryID: "github.com/example/repo",
		BaseRevision: baseRevision,
		Branch:       "synchestra/dispatch-1",
		Commit:       resultCommit,
		Summary:      "Updated dependencies",
		Validation: []dispatchcontract.ValidationEvidence{{
			Name: "go test", Command: "go test ./...", Status: dispatchcontract.ValidationPassed,
		}},
	}
	if err := result.Validate(); err == nil {
		t.Fatal("branch result without publication timestamp was accepted")
	}

	result.PublishedAt = time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	if err := result.Validate(); err != nil {
		t.Fatalf("complete branch result rejected: %v", err)
	}
}
