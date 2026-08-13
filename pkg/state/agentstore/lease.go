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
// primitive in this package: worktreeStore.Claim/Renew/Release/Handoff
// (worktree.go) delegate their exclusivity entirely to this type rather than
// duplicating the check-then-append race handling.
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

// leaseReclaimedPayload is task-3's TTL/expiry reclaim event
// (agent-coordination#ac:abandoned-run-is-resumable): a single audited event
// both invalidates the abandoned lease's fence and grants the new one, so
// there is no window in which the projection could show the resource as
// simultaneously free and held (the same "one event per compound state
// change" shape KindWorktreeReclaimed and KindLeaseTransferred use, and the
// same reasoning the package README's Open Questions note for why plain
// Claim already needs no separate transaction primitive here).
type leaseReclaimedPayload struct {
	Schema                string    `json:"schema"`
	LeaseID               string    `json:"lease_id"`
	Resource              string    `json:"resource"`
	HolderRunID           string    `json:"holder_run_id"`
	Epoch                 int64     `json:"authority_epoch"`
	Token                 string    `json:"fence_token"`
	AcquiredAt            time.Time `json:"acquired_at"`
	ExpiresAt             time.Time `json:"expires_at,omitempty"`
	SupersededLeaseID     string    `json:"superseded_lease_id"`
	SupersededHolderRunID string    `json:"superseded_holder_run_id"`
	SupersededExpiredAt   time.Time `json:"superseded_expired_at"`
}

// leaseTransferredPayload is task-3's voluntary-handoff event: the SAME
// lease ID moves to a new holder under a freshly minted fence token, unlike
// Reclaim (which mints a brand new lease ID for an abandoned resource).
type leaseTransferredPayload struct {
	Schema          string    `json:"schema"`
	LeaseID         string    `json:"lease_id"`
	FromHolderRunID string    `json:"from_holder_run_id"`
	ToHolderRunID   string    `json:"to_holder_run_id"`
	Epoch           int64     `json:"authority_epoch"`
	Token           string    `json:"fence_token"`
	TransferredAt   time.Time `json:"transferred_at"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
}

// leaseProjection is the deterministic in-memory fold of every lease event
// for one project, used both to answer Get/List-shaped reads and to decide
// whether Acquire may proceed.
type leaseProjection struct {
	byID       map[string]state.AuthorityLease
	released   map[string]bool
	byResource map[string]string // resource -> lease ID, only while active
}

// leaseEventKinds is every Kind foldLeaseProjection understands, shared by
// loadLeaseProjection/loadLeaseSnapshot so the two can never drift apart.
var leaseEventKinds = []string{KindLeaseAcquired, KindLeaseRenewed, KindLeaseReleased, KindLeaseReclaimed, KindLeaseTransferred}

func (s *Store) loadLeaseProjection(ctx context.Context) (leaseProjection, error) {
	events, err := s.loadEvents(ctx, leaseEventKinds...)
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
	head, headHash, events, err := s.snapshot(ctx, leaseEventKinds...)
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
			// A Renewed event for an ID the fold has already marked
			// released/reclaimed is never legitimate under the CAS-
			// protected Renew below (a release/reclaim that lands first
			// fences every later Renew attempt on that ID before it can
			// append) -- guarding here too keeps this fold correct even
			// against a corrupted or out-of-order event stream, rather
			// than silently resurrecting a dead lease.
			if lease, ok := proj.byID[payload.LeaseID]; ok && !proj.released[payload.LeaseID] {
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
		case KindLeaseReclaimed:
			var payload leaseReclaimedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return leaseProjection{}, fmt.Errorf("agentstore: decode %s: %w", KindLeaseReclaimed, err)
			}
			// Release the superseded (abandoned) lease first...
			if superseded, ok := proj.byID[payload.SupersededLeaseID]; ok {
				proj.released[payload.SupersededLeaseID] = true
				if proj.byResource[superseded.Resource] == payload.SupersededLeaseID {
					delete(proj.byResource, superseded.Resource)
				}
			}
			// ...then establish the new lease, exactly like Acquired does.
			lease := state.AuthorityLease{
				ID: payload.LeaseID, Resource: payload.Resource, HolderRunID: payload.HolderRunID,
				Fence:      state.LeaseFence{Epoch: payload.Epoch, Token: payload.Token},
				AcquiredAt: payload.AcquiredAt, RenewedAt: payload.AcquiredAt, ExpiresAt: payload.ExpiresAt,
			}
			proj.byID[lease.ID] = lease
			delete(proj.released, lease.ID)
			proj.byResource[lease.Resource] = lease.ID
		case KindLeaseTransferred:
			var payload leaseTransferredPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return leaseProjection{}, fmt.Errorf("agentstore: decode %s: %w", KindLeaseTransferred, err)
			}
			// Same guard as KindLeaseRenewed above: a Transferred event
			// never legitimately lands for an already-released/reclaimed
			// ID once Transfer is CAS-protected (below), but the fold
			// stays defensively correct either way.
			if lease, ok := proj.byID[payload.LeaseID]; ok && !proj.released[payload.LeaseID] {
				lease.HolderRunID = payload.ToHolderRunID
				lease.Fence = state.LeaseFence{Epoch: payload.Epoch, Token: payload.Token}
				lease.RenewedAt = payload.TransferredAt
				lease.ExpiresAt = payload.ExpiresAt
				proj.byID[payload.LeaseID] = lease
			}
		}
	}
	return proj, nil
}

// leaseExpired reports whether lease's recorded TTL deadline has passed as
// of now. A zero ExpiresAt (TTL<=0 at Acquire time) never expires, matching
// pre-task-3 behavior. See LeaseStore.Acquire's doc comment for the
// wall-clock caveats this basis carries.
func leaseExpired(lease state.AuthorityLease, now time.Time) bool {
	return !lease.ExpiresAt.IsZero() && !now.Before(lease.ExpiresAt)
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
		now := l.store.options.Now()
		var expiresAt time.Time
		if params.TTL > 0 {
			expiresAt = now.Add(params.TTL)
		}
		if existingID, held := proj.byResource[params.Resource]; held {
			existing := proj.byID[existingID]
			if !leaseExpired(existing, now) {
				return state.AuthorityLease{}, fmt.Errorf("agentstore: resource %q already leased by run %q: %w", params.Resource, existing.HolderRunID, state.ErrConflict)
			}
			// The current holder's TTL has elapsed and it was never
			// explicitly released: reclaim it for the new caller
			// (agent-coordination#ac:abandoned-run-is-resumable). One event
			// both fences the old holder's token and grants the new lease.
			leaseID := l.store.options.NewID()
			lease := state.AuthorityLease{
				ID: leaseID, Resource: params.Resource, HolderRunID: params.HolderRunID,
				Fence:      state.LeaseFence{Epoch: l.store.options.AuthorityEpoch, Token: l.store.options.NewID()},
				AcquiredAt: now, RenewedAt: now, ExpiresAt: expiresAt,
				Reclaimed: true, SupersededLeaseID: existing.ID,
			}
			payload := leaseReclaimedPayload{
				Schema: schemaLeaseReclaimedV1, LeaseID: lease.ID, Resource: lease.Resource, HolderRunID: lease.HolderRunID,
				Epoch: lease.Fence.Epoch, Token: lease.Fence.Token, AcquiredAt: now, ExpiresAt: expiresAt,
				SupersededLeaseID: existing.ID, SupersededHolderRunID: existing.HolderRunID, SupersededExpiredAt: existing.ExpiresAt,
			}
			if _, err := l.store.tryAppendAt(ctx, head, headHash, KindLeaseReclaimed, params.Resource, payload); err != nil {
				if errors.Is(err, errSequenceRace) {
					lastErr = err
					continue
				}
				return state.AuthorityLease{}, err
			}
			return lease, nil
		}
		leaseID := l.store.options.NewID()
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

// Renew re-derives the ENTIRE domain decision (projection + fence check)
// from a fresh, head-pinned snapshot on every retry attempt, exactly like
// Acquire above -- this is what makes it a real compare-and-swap rather than
// a stale read followed by a blind append. A plain "read once, then
// appendWithRetry" shape (what this used to do) is unsafe here: a concurrent
// TTL reclaim (Acquire) can commit between the read and the append, and a
// blind retry would re-append without ever noticing the lease it thinks it
// still holds was just fenced out from under it -- returning success to a
// now-dead holder (agent-coordination#ac:one-writer-claim-is-fenced's
// "resumed Renew after a concurrent reclaim must be refused", the double-
// writer this closes). See TestLeaseRenewRefusesConcurrentReclaimInFence
// CheckWindow for the deterministic reproduction.
func (l leaseStore) Renew(ctx context.Context, leaseID string, fence state.LeaseFence) (state.AuthorityLease, error) {
	var lastErr error
	for attempt := 0; attempt < maxAppendRetries; attempt++ {
		head, headHash, proj, err := l.store.loadLeaseSnapshot(ctx)
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
		l.store.fireTestSeam("renew", leaseID)
		now := l.store.options.Now()
		var expiresAt time.Time
		if !lease.ExpiresAt.IsZero() {
			expiresAt = now.Add(lease.ExpiresAt.Sub(lease.RenewedAt))
		}
		payload := leaseRenewedPayload{Schema: schemaLeaseRenewedV1, LeaseID: leaseID, RenewedAt: now, ExpiresAt: expiresAt}
		if _, err := l.store.tryAppendAt(ctx, head, headHash, KindLeaseRenewed, lease.Resource, payload); err != nil {
			if errors.Is(err, errSequenceRace) {
				lastErr = err
				continue
			}
			return state.AuthorityLease{}, err
		}
		lease.RenewedAt = now
		lease.ExpiresAt = expiresAt
		return lease, nil
	}
	return state.AuthorityLease{}, fmt.Errorf("agentstore: renew lease %q exhausted retries: %w", leaseID, lastErr)
}

// Release is Renew's CAS-protected twin: see Renew's doc comment for why a
// stale read followed by appendWithRetry is unsafe, and why every retry must
// re-derive the release decision (existence, already-released, fence match)
// from a fresh snapshot instead of trusting the first read.
func (l leaseStore) Release(ctx context.Context, leaseID string, fence state.LeaseFence) error {
	var lastErr error
	for attempt := 0; attempt < maxAppendRetries; attempt++ {
		head, headHash, proj, err := l.store.loadLeaseSnapshot(ctx)
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
		l.store.fireTestSeam("release", leaseID)
		payload := leaseReleasedPayload{Schema: schemaLeaseReleasedV1, LeaseID: leaseID, ReleasedAt: l.store.options.Now()}
		if _, err := l.store.tryAppendAt(ctx, head, headHash, KindLeaseReleased, lease.Resource, payload); err != nil {
			if errors.Is(err, errSequenceRace) {
				lastErr = err
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("agentstore: release lease %q exhausted retries: %w", leaseID, lastErr)
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

// Transfer proves the presented fence is still current and moves this SAME
// lease to toHolderRunID under a freshly minted fence token, in one audited
// event — the voluntary-handoff counterpart to Acquire's TTL-based reclaim.
// WorktreeStore.Handoff is built on this. Like Renew/Release above, every
// retry re-derives the fence decision from a fresh head-pinned snapshot
// (see Renew's doc comment): a concurrent TTL reclaim landing in the window
// between this call's fence check and its append must fence the transfer,
// not let it silently succeed against a lease that's already someone else's.
func (l leaseStore) Transfer(ctx context.Context, leaseID string, fence state.LeaseFence, toHolderRunID string) (state.AuthorityLease, error) {
	if strings.TrimSpace(toHolderRunID) == "" {
		return state.AuthorityLease{}, fmt.Errorf("agentstore: lease transfer needs a target holder run id")
	}
	var lastErr error
	for attempt := 0; attempt < maxAppendRetries; attempt++ {
		head, headHash, proj, err := l.store.loadLeaseSnapshot(ctx)
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
		l.store.fireTestSeam("transfer", leaseID)
		now := l.store.options.Now()
		var expiresAt time.Time
		if !lease.ExpiresAt.IsZero() {
			expiresAt = now.Add(lease.ExpiresAt.Sub(lease.RenewedAt))
		}
		newFence := state.LeaseFence{Epoch: l.store.options.AuthorityEpoch, Token: l.store.options.NewID()}
		payload := leaseTransferredPayload{
			Schema: schemaLeaseTransferredV1, LeaseID: leaseID, FromHolderRunID: lease.HolderRunID, ToHolderRunID: toHolderRunID,
			Epoch: newFence.Epoch, Token: newFence.Token, TransferredAt: now, ExpiresAt: expiresAt,
		}
		if _, err := l.store.tryAppendAt(ctx, head, headHash, KindLeaseTransferred, lease.Resource, payload); err != nil {
			if errors.Is(err, errSequenceRace) {
				lastErr = err
				continue
			}
			return state.AuthorityLease{}, err
		}
		lease.HolderRunID = toHolderRunID
		lease.Fence = newFence
		lease.RenewedAt = now
		lease.ExpiresAt = expiresAt
		return lease, nil
	}
	return state.AuthorityLease{}, fmt.Errorf("agentstore: transfer lease %q exhausted retries: %w", leaseID, lastErr)
}
