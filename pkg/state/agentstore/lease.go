package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

// leaseStore implements state.LeaseStore. It is the sole uniqueness/fencing
// primitive in this package: worktreeStore.Claim/Renew/Release (worktree.go)
// delegate their exclusivity entirely to this type rather than duplicating
// the check-then-append race handling.
type leaseStore struct{ store *Store }

var _ state.LeaseStore = leaseStore{}

type leaseAcquiredPayload struct {
	Schema      string    `json:"schema"`
	LeaseID     string    `json:"lease_id"`
	Resource    string    `json:"resource"`
	HolderRunID string    `json:"holder_run_id"`
	Epoch       int64     `json:"authority_epoch"`
	Token       string    `json:"fence_token"`
	AcquiredAt  time.Time `json:"acquired_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

type leaseRenewedPayload struct {
	Schema    string    `json:"schema"`
	LeaseID   string    `json:"lease_id"`
	RenewedAt time.Time `json:"renewed_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type leaseReleasedPayload struct {
	Schema     string    `json:"schema"`
	LeaseID    string    `json:"lease_id"`
	ReleasedAt time.Time `json:"released_at"`
}

// leaseProjection is the deterministic in-memory fold of every lease event
// for one project, used both to answer Get/List-shaped reads and to decide
// whether Acquire may proceed.
type leaseProjection struct {
	byID       map[string]state.AuthorityLease
	released   map[string]bool
	byResource map[string]string // resource -> lease ID, only while active
}

func (s *Store) loadLeaseProjection(ctx context.Context) (leaseProjection, error) {
	events, err := s.loadEvents(ctx, KindLeaseAcquired, KindLeaseRenewed, KindLeaseReleased)
	if err != nil {
		return leaseProjection{}, err
	}
	return foldLeaseProjection(events)
}

// loadLeaseSnapshot pairs the lease projection with the exact journal head it
// was folded from, so Acquire can append pinned to that same observation
// (see Store.snapshot's doc comment for why this closes the TOCTOU gap a
// plain projection-then-append would leave open).
func (s *Store) loadLeaseSnapshot(ctx context.Context) (replication.Cursor, string, leaseProjection, error) {
	head, headHash, events, err := s.snapshot(ctx, KindLeaseAcquired, KindLeaseRenewed, KindLeaseReleased)
	if err != nil {
		return replication.Cursor{}, "", leaseProjection{}, err
	}
	proj, err := foldLeaseProjection(events)
	if err != nil {
		return replication.Cursor{}, "", leaseProjection{}, err
	}
	return head, headHash, proj, nil
}

func foldLeaseProjection(events []replication.Event) (leaseProjection, error) {
	proj := leaseProjection{byID: map[string]state.AuthorityLease{}, released: map[string]bool{}, byResource: map[string]string{}}
	for _, event := range events {
		switch event.Kind {
		case KindLeaseAcquired:
			var payload leaseAcquiredPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return leaseProjection{}, fmt.Errorf("agentstore: decode %s: %w", KindLeaseAcquired, err)
			}
			lease := state.AuthorityLease{
				ID: payload.LeaseID, Resource: payload.Resource, HolderRunID: payload.HolderRunID,
				Fence:      state.LeaseFence{Epoch: payload.Epoch, Token: payload.Token},
				AcquiredAt: payload.AcquiredAt, RenewedAt: payload.AcquiredAt, ExpiresAt: payload.ExpiresAt,
			}
			proj.byID[lease.ID] = lease
			delete(proj.released, lease.ID)
			proj.byResource[lease.Resource] = lease.ID
		case KindLeaseRenewed:
			var payload leaseRenewedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return leaseProjection{}, fmt.Errorf("agentstore: decode %s: %w", KindLeaseRenewed, err)
			}
			if lease, ok := proj.byID[payload.LeaseID]; ok {
				lease.RenewedAt = payload.RenewedAt
				lease.ExpiresAt = payload.ExpiresAt
				proj.byID[payload.LeaseID] = lease
			}
		case KindLeaseReleased:
			var payload leaseReleasedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return leaseProjection{}, fmt.Errorf("agentstore: decode %s: %w", KindLeaseReleased, err)
			}
			if lease, ok := proj.byID[payload.LeaseID]; ok {
				proj.released[payload.LeaseID] = true
				if proj.byResource[lease.Resource] == payload.LeaseID {
					delete(proj.byResource, lease.Resource)
				}
			}
		}
	}
	return proj, nil
}

func (l leaseStore) Acquire(ctx context.Context, params state.LeaseAcquireParams) (state.AuthorityLease, error) {
	if strings.TrimSpace(params.Resource) == "" {
		return state.AuthorityLease{}, fmt.Errorf("agentstore: lease resource is required")
	}
	if strings.TrimSpace(params.HolderRunID) == "" {
		return state.AuthorityLease{}, fmt.Errorf("agentstore: lease holder run id is required")
	}
	var lastErr error
	for attempt := 0; attempt < maxAppendRetries; attempt++ {
		head, headHash, proj, err := l.store.loadLeaseSnapshot(ctx)
		if err != nil {
			return state.AuthorityLease{}, err
		}
		if existingID, held := proj.byResource[params.Resource]; held {
			existing := proj.byID[existingID]
			return state.AuthorityLease{}, fmt.Errorf("agentstore: resource %q already leased by run %q: %w", params.Resource, existing.HolderRunID, state.ErrConflict)
		}
		now := l.store.options.Now()
		leaseID := l.store.options.NewID()
		var expiresAt time.Time
		if params.TTL > 0 {
			expiresAt = now.Add(params.TTL)
		}
		lease := state.AuthorityLease{
			ID: leaseID, Resource: params.Resource, HolderRunID: params.HolderRunID,
			Fence:      state.LeaseFence{Epoch: l.store.options.AuthorityEpoch, Token: l.store.options.NewID()},
			AcquiredAt: now, RenewedAt: now, ExpiresAt: expiresAt,
		}
		payload := leaseAcquiredPayload{
			Schema: schemaLeaseAcquiredV1, LeaseID: lease.ID, Resource: lease.Resource, HolderRunID: lease.HolderRunID,
			Epoch: lease.Fence.Epoch, Token: lease.Fence.Token, AcquiredAt: now, ExpiresAt: expiresAt,
		}
		// Pinned to the exact snapshot the uniqueness check above was decided
		// from: if any event — for this resource or any other — landed since
		// that snapshot, this Append fails and the loop retries against a
		// fresh snapshot rather than blindly succeeding at "the next slot".
		if _, err := l.store.tryAppendAt(ctx, head, headHash, KindLeaseAcquired, params.Resource, payload); err != nil {
			if errors.Is(err, errSequenceRace) {
				lastErr = err
				continue
			}
			return state.AuthorityLease{}, err
		}
		return lease, nil
	}
	return state.AuthorityLease{}, fmt.Errorf("agentstore: acquire lease for %q exhausted retries: %w", params.Resource, lastErr)
}

func (l leaseStore) Renew(ctx context.Context, leaseID string, fence state.LeaseFence) (state.AuthorityLease, error) {
	proj, err := l.store.loadLeaseProjection(ctx)
	if err != nil {
		return state.AuthorityLease{}, err
	}
	lease, ok := proj.byID[leaseID]
	if !ok {
		return state.AuthorityLease{}, fmt.Errorf("agentstore: lease %q: %w", leaseID, state.ErrNotFound)
	}
	if proj.released[leaseID] {
		return state.AuthorityLease{}, fmt.Errorf("agentstore: lease %q already released: %w", leaseID, state.ErrLeaseFenced)
	}
	if lease.Fence != fence {
		return state.AuthorityLease{}, fmt.Errorf("agentstore: lease %q fence %+v does not match current %+v: %w", leaseID, fence, lease.Fence, state.ErrLeaseFenced)
	}
	now := l.store.options.Now()
	var expiresAt time.Time
	if !lease.ExpiresAt.IsZero() {
		expiresAt = now.Add(lease.ExpiresAt.Sub(lease.RenewedAt))
	}
	payload := leaseRenewedPayload{Schema: schemaLeaseRenewedV1, LeaseID: leaseID, RenewedAt: now, ExpiresAt: expiresAt}
	if _, err := l.store.appendWithRetry(ctx, KindLeaseRenewed, lease.Resource, payload); err != nil {
		return state.AuthorityLease{}, err
	}
	lease.RenewedAt = now
	lease.ExpiresAt = expiresAt
	return lease, nil
}

func (l leaseStore) Release(ctx context.Context, leaseID string, fence state.LeaseFence) error {
	proj, err := l.store.loadLeaseProjection(ctx)
	if err != nil {
		return err
	}
	lease, ok := proj.byID[leaseID]
	if !ok {
		return fmt.Errorf("agentstore: lease %q: %w", leaseID, state.ErrNotFound)
	}
	if proj.released[leaseID] {
		return nil // already released: releasing twice is not an error.
	}
	if lease.Fence != fence {
		return fmt.Errorf("agentstore: lease %q fence %+v does not match current %+v: %w", leaseID, fence, lease.Fence, state.ErrLeaseFenced)
	}
	payload := leaseReleasedPayload{Schema: schemaLeaseReleasedV1, LeaseID: leaseID, ReleasedAt: l.store.options.Now()}
	_, err = l.store.appendWithRetry(ctx, KindLeaseReleased, lease.Resource, payload)
	return err
}

func (l leaseStore) Get(ctx context.Context, resource string) (state.AuthorityLease, error) {
	proj, err := l.store.loadLeaseProjection(ctx)
	if err != nil {
		return state.AuthorityLease{}, err
	}
	leaseID, ok := proj.byResource[resource]
	if !ok {
		return state.AuthorityLease{}, fmt.Errorf("agentstore: resource %q: %w", resource, state.ErrNotFound)
	}
	return proj.byID[leaseID], nil
}
