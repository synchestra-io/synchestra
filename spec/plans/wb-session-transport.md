---
format: https://specscore.md/plan-specification
status: Approved
---

# Plan: WB Session Transport implementation plan

**Status:** Executing
**Source Feature:** wb-session-transport
**Date:** 2026-08-25
**Owner:** codex
**Supersedes:** —
**Parent:** wb:agent-session-move

## Summary

Implement a typed Synchestra courier for WB handoffs and messages while keeping
the public surface free of arbitrary command semantics. The current CLI and
dispatch contract carry the invocation; the sibling VM runner adds the fixed
handlers and returns an integrity-checked WB receipt artifact.

## Approach

Expose a clean `runner invoke` API backed internally by a typed, validated
dispatch-v1 compatibility envelope so existing Hub scheduling, leasing,
status, logs, retry, and cancellation remain unchanged. Add fixed handler
execution and receipt publication in `synchestra-vm`, then prove the path with
fake WB and temporary Git fixtures before the live WB plan selects this
courier. A first-class dispatch result union and a generalized handler engine
for every runner implementation are deferred because the VM-first receipt
artifact satisfies the MVP without a wire-version break.

## Tasks

### Task 1: Define the typed invocation compatibility contract

**Verifies:** wb-session-transport#ac:arbitrary-command-input-is-rejected
**Status:** complete

Add canonical `HandlerInvocation` construction, parsing, digesting, size
bounds, reserved project-context encoding, closed WB handler names, synthetic
selector/capability helpers, handoff-ID idempotency-key derivation, and
validation tests in `pkg/dispatch-contract`. Request data must never supply
executable or shell argv, and ordinary dispatch contracts must remain
byte-compatible.

### Task 2: Add `synchestra runner invoke` and lifecycle observation

**Verifies:** wb-session-transport#ac:lifecycle-controls-apply-to-wb-invocations
**Status:** complete

Implement required runner, handler, and `@payload-file` flags, bounded JSON
loading, immutable repository evidence, create/status output, and stable JSON
fields through the existing dispatch client. Reuse the terminal attempt result
for a repeated handoff ID with the same digest and reject the same ID with a
different digest at dispatch creation. Prove invocation creation leaves the
caller checkout untouched and that existing status, logs, retry, and cancel
verbs observe the same durable attempt lifecycle.

### Task 3: Execute fixed WB accept and message handlers on the VM runner

**Verifies:** wb-session-transport#ac:typed-wb-handoff-reaches-fixed-handler, wb-session-transport#ac:message-handler-preserves-courier-boundary
**Status:** complete

In a dedicated `synchestra-vm` feature worktree, advertise the two handler
capabilities, route only registered names, stage validated payloads privately,
and call fixed WB receive/message argv with `exec.CommandContext`. Validate the
WB receipt artifact without parsing handover or lineage semantics, and preserve
lease loss, cancellation, cleanup, and log-redaction behavior.

### Task 4: Prove idempotent receipt delivery end to end

**Verifies:** wb-session-transport#ac:retry-returns-one-wb-receipt
**Status:** planning

Add a fake-WB vertical test spanning CLI intent, durable dispatch attempt, VM
handler, receipt branch/artifact, and result observation. Assert a lost-response
retry returns the same successor receipt without a second active lease or
handler launch, while a digest conflict fails and ordinary dispatch remains
unchanged.

## Open Questions

1. After the VM-first MVP is proven, a follow-on dispatch-contract Feature can
   decide whether opaque inline handler results justify a new wire version over
   the existing immutable receipt artifact.

---
*This document follows the https://specscore.md/plan-specification*
