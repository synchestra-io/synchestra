package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// defaultWorktreeLeaseTTL bounds how long a worktree claim's underlying
// lease is considered live before an operator/refresh policy (task-3 scope)
// must renew it. It is a data default, not a lifecycle decision.
const defaultWorktreeLeaseTTL = time.Hour

// worktreeStore implements state.WorktreeStore. Exclusivity ("exactly one
// concurrent writer per worktree/branch", agent-coordination#ac:one-writer-
// claim-is-fenced) is delegated entirely to leaseStore — this type only
// records the Git facts (path/branch/base/head/scope) alongside the lease it
// acquires, and never re-implements check-then-append race handling.
type worktreeStore struct{ store *Store }

var _ state.WorktreeStore = worktreeStore{}

// worktreeResourceKey is the exclusivity key: one live claim per
// project+repository+branch, matching "The MVP permits one concurrent writer
// per worktree and branch" (spec/features/agent-coordination).
func worktreeResourceKey(projectID, repositoryID, branch string) string {
	return "worktree:" + projectID + ":" + repositoryID + ":" + branch
}

type worktreeClaimedPayload struct {
	Schema       string    `json:"schema"`
	ClaimID      string    `json:"claim_id"`
	ProjectID    string    `json:"project_id"`
	RepositoryID string    `json:"repository_id"`
	RunID        string    `json:"run_id"`
	WorktreePath string    `json:"worktree_path"`
	Branch       string    `json:"branch"`
	TargetRef    string    `json:"target_ref,omitempty"`
	BaseSHA      string    `json:"base_sha,omitempty"`
	HeadSHA      string    `json:"head_sha,omitempty"`
	ScopeAreas   []string  `json:"scope_areas,omitempty"`
	FenceEpoch   int64     `json:"fence_epoch"`
	FenceToken   string    `json:"fence_token"`
	ClaimedAt    time.Time `json:"claimed_at"`
}

type worktreeRenewedPayload struct {
	Schema     string    `json:"schema"`
	ClaimID    string    `json:"claim_id"`
	FenceEpoch int64     `json:"fence_epoch"`
	FenceToken string    `json:"fence_token"`
	RenewedAt  time.Time `json:"renewed_at"`
}

type worktreeReleasedPayload struct {
	Schema     string    `json:"schema"`
	ClaimID    string    `json:"claim_id"`
	ReleasedAt time.Time `json:"released_at"`
}

func (s *Store) loadWorktreeProjection(ctx context.Context) (map[string]state.WorktreeClaim, error) {
	events, err := s.loadEvents(ctx, KindWorktreeClaimed, KindWorktreeRenewed, KindWorktreeReleased)
	if err != nil {
		return nil, err
	}
	claims := map[string]state.WorktreeClaim{}
	for _, event := range events {
		switch event.Kind {
		case KindWorktreeClaimed:
			var payload worktreeClaimedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("agentstore: decode %s: %w", KindWorktreeClaimed, err)
			}
			claims[payload.ClaimID] = state.WorktreeClaim{
				ID: payload.ClaimID, ProjectID: payload.ProjectID, RepositoryID: payload.RepositoryID, RunID: payload.RunID,
				WorktreePath: payload.WorktreePath, Branch: payload.Branch, TargetRef: payload.TargetRef,
				BaseSHA: payload.BaseSHA, HeadSHA: payload.HeadSHA, ScopeAreas: payload.ScopeAreas,
				Fence:     state.LeaseFence{Epoch: payload.FenceEpoch, Token: payload.FenceToken},
				ClaimedAt: payload.ClaimedAt, RenewedAt: payload.ClaimedAt,
			}
		case KindWorktreeRenewed:
			var payload worktreeRenewedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("agentstore: decode %s: %w", KindWorktreeRenewed, err)
			}
			if claim, ok := claims[payload.ClaimID]; ok {
				claim.Fence = state.LeaseFence{Epoch: payload.FenceEpoch, Token: payload.FenceToken}
				claim.RenewedAt = payload.RenewedAt
				claims[payload.ClaimID] = claim
			}
		case KindWorktreeReleased:
			var payload worktreeReleasedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("agentstore: decode %s: %w", KindWorktreeReleased, err)
			}
			if claim, ok := claims[payload.ClaimID]; ok {
				releasedAt := payload.ReleasedAt
				claim.ReleasedAt = &releasedAt
				claims[payload.ClaimID] = claim
			}
		}
	}
	return claims, nil
}

func (w worktreeStore) Claim(ctx context.Context, params state.WorktreeClaimParams) (state.WorktreeClaim, error) {
	if err := validateWorktreeClaimParams(params); err != nil {
		return state.WorktreeClaim{}, err
	}
	resource := worktreeResourceKey(params.ProjectID, params.RepositoryID, params.Branch)
	lease, err := w.store.Lease().Acquire(ctx, state.LeaseAcquireParams{
		Resource: resource, HolderRunID: params.RunID, TTL: defaultWorktreeLeaseTTL,
	})
	if err != nil {
		return state.WorktreeClaim{}, err
	}
	now := w.store.options.Now()
	scope := append([]string(nil), params.ScopeAreas...)
	claim := state.WorktreeClaim{
		ID: lease.ID, ProjectID: params.ProjectID, RepositoryID: params.RepositoryID, RunID: params.RunID,
		WorktreePath: params.WorktreePath, Branch: params.Branch, TargetRef: params.TargetRef,
		BaseSHA: params.BaseSHA, HeadSHA: params.HeadSHA, ScopeAreas: scope,
		Fence: lease.Fence, ClaimedAt: now, RenewedAt: now,
	}
	payload := worktreeClaimedPayload{
		Schema: schemaWorktreeClaimedV1, ClaimID: claim.ID, ProjectID: params.ProjectID, RepositoryID: params.RepositoryID,
		RunID: params.RunID, WorktreePath: params.WorktreePath, Branch: params.Branch, TargetRef: params.TargetRef,
		BaseSHA: params.BaseSHA, HeadSHA: params.HeadSHA, ScopeAreas: scope,
		FenceEpoch: lease.Fence.Epoch, FenceToken: lease.Fence.Token, ClaimedAt: now,
	}
	// The lease is already the durable, exclusive grant; this event records
	// the Git facts alongside it and only needs benign-race retry, not a
	// fresh uniqueness check (see appendWithRetry's doc comment).
	if _, err := w.store.appendWithRetry(ctx, KindWorktreeClaimed, params.RepositoryID+":"+params.Branch, payload); err != nil {
		return state.WorktreeClaim{}, err
	}
	return claim, nil
}

func (w worktreeStore) Renew(ctx context.Context, claimID string, fence state.LeaseFence) (state.WorktreeClaim, error) {
	claim, err := w.Get(ctx, claimID)
	if err != nil {
		return state.WorktreeClaim{}, err
	}
	if claim.ReleasedAt != nil {
		return state.WorktreeClaim{}, fmt.Errorf("agentstore: worktree claim %q already released: %w", claimID, state.ErrLeaseFenced)
	}
	lease, err := w.store.Lease().Renew(ctx, claimID, fence)
	if err != nil {
		return state.WorktreeClaim{}, err
	}
	payload := worktreeRenewedPayload{
		Schema: schemaWorktreeRenewedV1, ClaimID: claimID, FenceEpoch: lease.Fence.Epoch, FenceToken: lease.Fence.Token,
		RenewedAt: lease.RenewedAt,
	}
	if _, err := w.store.appendWithRetry(ctx, KindWorktreeRenewed, claim.RepositoryID+":"+claim.Branch, payload); err != nil {
		return state.WorktreeClaim{}, err
	}
	claim.Fence = lease.Fence
	claim.RenewedAt = lease.RenewedAt
	return claim, nil
}

func (w worktreeStore) Release(ctx context.Context, claimID string, fence state.LeaseFence) error {
	claim, err := w.Get(ctx, claimID)
	if err != nil {
		return err
	}
	if claim.ReleasedAt != nil {
		return nil // already released: releasing twice is not an error.
	}
	if err := w.store.Lease().Release(ctx, claimID, fence); err != nil {
		return err
	}
	payload := worktreeReleasedPayload{Schema: schemaWorktreeReleasedV1, ClaimID: claimID, ReleasedAt: w.store.options.Now()}
	_, err = w.store.appendWithRetry(ctx, KindWorktreeReleased, claim.RepositoryID+":"+claim.Branch, payload)
	return err
}

func (w worktreeStore) Get(ctx context.Context, claimID string) (state.WorktreeClaim, error) {
	claims, err := w.store.loadWorktreeProjection(ctx)
	if err != nil {
		return state.WorktreeClaim{}, err
	}
	claim, ok := claims[claimID]
	if !ok {
		return state.WorktreeClaim{}, fmt.Errorf("agentstore: worktree claim %q: %w", claimID, state.ErrNotFound)
	}
	return claim, nil
}

func (w worktreeStore) List(ctx context.Context, filter state.WorktreeFilter) ([]state.WorktreeClaim, error) {
	claims, err := w.store.loadWorktreeProjection(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]state.WorktreeClaim, 0, len(claims))
	for _, claim := range claims {
		if filter.ProjectID != "" && claim.ProjectID != filter.ProjectID {
			continue
		}
		if filter.RepositoryID != "" && claim.RepositoryID != filter.RepositoryID {
			continue
		}
		if filter.RunID != "" && claim.RunID != filter.RunID {
			continue
		}
		if filter.ActiveOnly && claim.ReleasedAt != nil {
			continue
		}
		result = append(result, claim)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ClaimedAt.Before(result[j].ClaimedAt) })
	return result, nil
}

func validateWorktreeClaimParams(params state.WorktreeClaimParams) error {
	if strings.TrimSpace(params.ProjectID) == "" || strings.TrimSpace(params.RepositoryID) == "" ||
		strings.TrimSpace(params.RunID) == "" || strings.TrimSpace(params.Branch) == "" {
		return fmt.Errorf("agentstore: worktree claim needs project id, repository id, run id, and branch")
	}
	return nil
}
