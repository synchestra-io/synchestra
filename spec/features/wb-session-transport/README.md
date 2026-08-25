---
format: https://specscore.md/feature-specification
status: Under Review
---

# Feature: WB Session Transport

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/wb-session-transport?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/wb-session-transport?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/wb-session-transport?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/wb-session-transport?op=request-change) |
**Status:** Under Review
**Source Ideas:** —
**Supersedes:** —

## Summary

Accept typed WB session handoff requests through Synchestra durable runner infrastructure without exposing arbitrary remote shell execution.

## Problem

WB needs more than one courier for cross-machine agent handoffs. SSH provides a
direct reference path, while Synchestra can provide durable scheduling,
leases, retry, cancellation, and observability. Synchestra's existing dispatch
surface is repository-job oriented, however, and exposing an arbitrary command
runner would weaken both products' ownership boundaries and security.

Synchestra needs a narrow typed invocation seam that carries an opaque WB
payload to a registered fixed handler and returns its receipt. WB remains the
owner of Git checkpointing, handover contents, session identity, target tmux
startup, and chain of custody.

## Behavior

### Typed invocation contract

#### REQ: typed-runner-invocation

The CLI MUST expose a versioned typed invocation such as
`synchestra runner invoke --runner <id> --handler <name> --payload <file>`.
The request MUST record invocation ID, handler name and version, target runner,
payload digest and size, creation time, attempt state, and optional deadline.
Payload bytes are opaque to Synchestra.

#### REQ: fixed-wb-handlers

The MVP runner MUST support only explicitly registered WB handler names
`wb.session.accept.v1` and `wb.session.message.v1`. Each handler MUST execute a
configured argv template whose executable and subcommand are operator-owned;
payload fields MUST NOT select an executable, inject shell syntax, or become an
arbitrary command line. The receive handler invokes the installed fixed WB
receiver boundary, and the message handler invokes the fixed WB message
receiver boundary.

### Durable delivery

#### REQ: reuse-runner-lifecycle

WB invocations MUST reuse the existing runner's durable queue, eligibility,
lease, heartbeat, retry, cancellation, logs, and terminal-result lifecycle
rather than create a second scheduler. A handler attempt MUST carry the same
runner fencing and stale-owner rejection rules as other runner work.

#### REQ: idempotent-handoff-delivery

`wb.session.accept.v1` MUST use the WB handoff ID as its idempotency key. At
most one attempt may actively own an invocation, and redelivery of an already
completed invocation MUST return the stored handler receipt without launching
the handler again. Reusing an ID with a different payload digest MUST fail.

#### REQ: typed-handler-receipt

A successful invocation MUST persist and return an integrity-checked reference
to the handler's structured receipt without interpreting WB fields. The result
MUST include invocation ID, handler, runner, attempt, payload digest,
start/completion times, terminal status, and either the opaque WB receipt or an
immutable artifact reference from which it can be retrieved. Status and logs
MUST be queryable through the existing runner observation surface.

### Safety and ownership

#### REQ: bounded-payload-and-redaction

The service MUST enforce configured payload and log-size limits, verify the
declared digest before handler execution, stage payloads in a private temporary
file, remove them after terminal retention permits, and avoid logging payload
contents or credentials. Authentication and runner authorization MUST be the
same as other targeted runner invocations.

#### REQ: wb-remains-protocol-owner

Synchestra MUST NOT parse the handover Markdown, inspect or mutate the target
Git repository, allocate WB session IDs, start tmux directly, transfer Work Log
claims, or decide which harness runs. Its responsibility ends at delivering
bytes to the configured fixed handler and returning a durable receipt.

## Architecture

The typed invocation is a small adapter over the existing runner lifecycle:

```text
WB courier -> runner invocation record -> eligible leased runner
           -> fixed handler registry -> wb session receive/message
           <- opaque handler receipt <- durable terminal result
```

Handler registration is operator configuration and code, not request data.
This keeps Synchestra useful as a courier without turning it into WB or a
general remote-shell service. The MVP MAY encode the typed invocation inside a
reserved, validated dispatch-v1 compatibility envelope and return a receipt
branch through the existing branch-result contract; that rolling-compatible
representation is an implementation detail and MUST NOT leak anonymous map,
synthetic agent, or branch-result semantics into `runner invoke` callers.

## Acceptance Criteria

### AC: typed-wb-handoff-reaches-fixed-handler

**Requirements:** wb-session-transport#req:typed-runner-invocation, wb-session-transport#req:fixed-wb-handlers, wb-session-transport#req:typed-handler-receipt

Scenario: Invoke the registered WB receive handler
Given an authorized caller, an eligible runner, and a valid WB handoff payload file
When the caller invokes `wb.session.accept.v1` for that runner
Then the durable attempt executes only the configured WB receive argv, verifies the payload digest, and returns a structured terminal result containing the opaque WB receipt or its immutable artifact reference

### AC: arbitrary-command-input-is-rejected

**Requirements:** wb-session-transport#req:fixed-wb-handlers, wb-session-transport#req:bounded-payload-and-redaction, wb-session-transport#req:wb-remains-protocol-owner

Scenario: Reject a request-controlled executable
Given a request with an unknown handler or payload fields containing shell syntax and command names
When the runner validates the invocation
Then it refuses before execution, writes no payload content to logs, and never treats request data as an executable or shell command

### AC: retry-returns-one-wb-receipt

**Requirements:** wb-session-transport#req:reuse-runner-lifecycle, wb-session-transport#req:idempotent-handoff-delivery, wb-session-transport#req:typed-handler-receipt

Scenario: Redeliver after a lost response
Given a WB handoff invocation completed but its response was lost
When the caller submits the same handoff ID and payload digest again
Then Synchestra returns the stored terminal receipt without a second active lease or a second handler launch, while a different digest for that ID is rejected

### AC: lifecycle-controls-apply-to-wb-invocations

**Requirements:** wb-session-transport#req:reuse-runner-lifecycle, wb-session-transport#req:bounded-payload-and-redaction

Scenario: Observe and cancel a queued WB invocation
Given a queued or running WB invocation
When the caller uses the existing runner status, logs, or cancellation operations
Then those operations expose or change the same durable attempt lifecycle and fencing rules used by other runner work without exposing the handover payload

### AC: message-handler-preserves-courier-boundary

**Requirements:** wb-session-transport#req:fixed-wb-handlers, wb-session-transport#req:typed-handler-receipt, wb-session-transport#req:wb-remains-protocol-owner

Scenario: Deliver a follow-up message to WB
Given a completed WB handoff and an authorized typed message payload
When the caller invokes `wb.session.message.v1`
Then the configured WB message receiver handles the payload and returns its opaque receipt while Synchestra neither locates tmux nor interprets session lineage

## Rehearse Integration

Every acceptance criterion has a deterministic CLI, queue, data, process, or
log surface. Pending scenario stubs live under `_tests/` and should use a fake
fixed handler plus the existing runner test harness.

## Not Doing

- A general arbitrary-command runner.
- A second queue, lease system, or WB-specific scheduler.
- Parsing WB handovers or owning WB session, Git, tmux, or Work Log semantics.
- Choosing a model or AI harness for WB.
- SSH transport; SSH remains WB's independent reference courier.

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
