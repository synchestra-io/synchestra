# Dispatch Contract

This package is the frozen `synchestra.dispatch.v1` contract for Remote Task Dispatch. Its Go structures and JSON tags are the normative request, response, and durable-record schemas shared by `synchestra`, `synchestra-cloud`, `synchestra-vm`, `synchestra-servers`, and `ai-plugin-synchestra`.

The package intentionally contains no scheduler, HTTP routing, Firestore, repository execution, or agent logic. Component implementations import these types rather than defining parallel dispatch schemas.

## Versioning

- Every top-level request and durable `Dispatch`/`Attempt` carries `protocol_version: "synchestra.dispatch.v1"`.
- Additive optional fields may remain v1.
- Removing a field, changing a field's meaning, changing a required invariant, or changing lease ownership semantics requires a new major protocol value.
- MVP peers reject unsupported versions with `INCOMPATIBLE_PROTOCOL`; they do not guess or silently downgrade.

## Frozen Invariants

- A Dispatch holds immutable intent; an Attempt holds one claim/execution history entry; a Session identifies one concrete agent process.
- The repository base is a full immutable Git object ID. Symbolic branches are audit metadata only.
- Ad-hoc prompts do not require Tasks. SpecScore targets carry an immutable target revision and content hash.
- One authenticated worker owns an Attempt through `(attempt_id, worker_id, lease_generation)`. Reassignment increments the generation, making stale heartbeats and terminal writes invalid.
- Retrying appends a new queued Attempt and preserves all history. Claim transitions that exact queued Attempt to leased; implementations never persist a zero-value Attempt status.
- Exact selectors reject by default. Fallback is allowed only when the request explicitly records an allow-list and reason.
- The scheduler resolves routing during claim. Each leased Attempt already contains frozen requested and resolved profile, agent, model, effort, mapping version, and routing reason; the worker cannot rewrite that decision when it starts the Session.
- Attempt and Dispatch records reference one canonical log stream; they do not duplicate raw log text.
- Log messages and bounded validation summaries are redacted before submission. Clone URLs never contain credentials; persisted credential fields are references only.
- Success requires a `synchestra/<dispatch-id>` review branch, immutable commit, and validation evidence. Workers never merge or deploy.
- Cancellation or ownership loss prevents later success publication.

## Typed Handler Compatibility Envelope

`HandlerInvocation` is an additive adapter over dispatch v1; it does not add or
change fields on the frozen dispatch structures. `EncodeHandlerInvocation`
stores its canonical JSON in one reserved ad-hoc `project_context` entry, and
`ParseHandlerInvocation` distinguishes that entry from ordinary dispatch work.

- The only MVP handler names are `wb.session.accept.v1` and
  `wb.session.message.v1`.
- Payloads are opaque byte slices, limited to 1 MiB and protected by an exact
  SHA-256 digest and byte count.
- The invocation schema has no executable, shell, argv, or environment fields.
  Unknown envelope fields and unknown handler names are rejected before a
  caller can route them to a worker.
- Synthetic scheduler selectors and worker capabilities are created only by
  the typed helpers in this package. Request payload data never contributes to
  either value.
- `WBHandoffIdempotencyKey` derives a bounded deterministic dispatch key while
  keeping the caller's raw WB handoff ID out of the key.

## Existing-Field Reconciliation

| Existing field/surface | Dispatch v1 decision |
|---|---|
| Runner session `agent` | Persisted as both requested and resolved agent; immutable per Attempt. |
| Runner `small|medium|large` model sizes | User-facing profiles become `fast|balanced|large`; legacy `small|medium` are migration inputs, not v1 profile values. |
| Concrete session `model` | Stored in `ResolvedExecution.Model`; never inferred from profile after leasing. |
| Session/runner `effort` | Stored separately from model and propagated unchanged to the adapter. |
| Host/runner IDs | `WorkerIdentity` carries registered `host_id`, unique `worker_id`, and optional logical `runner_id`. |
| Repository `branch` | Becomes optional `base_ref` audit metadata; `base_revision` is the execution authority. |
| Project workspace paths | Dispatch carries repository identity, revision, and optional subdirectory; the execution adapter returns the canonical worktree root explicitly. |
| Session messages/logs | `LogReference` points to one append-only stream; Dispatch does not become a transcript store. |

## Operation Families

- Caller: create, inspect, logs, cancel, retry.
- Worker: claim, start, heartbeat, append logs, complete, fail, acknowledge cancellation.
- Claim is idempotent per worker `request_id`; all other mutations are idempotent per `operation_id`.
- Route spelling and authentication middleware are implementation-owned. The JSON bodies and lifecycle invariants are not.

## Open Questions

None at this time.
