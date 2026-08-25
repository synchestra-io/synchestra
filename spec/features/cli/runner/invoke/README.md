---
format: https://specscore.md/feature-specification
status: In Progress
---

# Feature: CLI Runner Invoke

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/runner/invoke?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/runner/invoke?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/runner/invoke?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/runner/invoke?op=request-change) |
**Status:** In Progress
**Source Ideas:** —

## Summary

Invokes one code-registered runner handler with opaque, bounded JSON bytes while
reusing the existing durable dispatch, attempt, lease, log, retry, and
cancellation lifecycle.

## Problem

WB needs a durable courier for handoffs and messages, but a request-controlled
command runner would give remote payloads authority over executable selection.
The CLI therefore needs a narrow typed surface that records safe routing and
integrity evidence without interpreting or printing WB payload fields.

## Contents

| Directory | Description |
|---|---|
| [_args/](_args/README.md) | Command-specific positional argument and flag contracts |

### _args

Defines the required target runner, registered handler, caller-owned invocation
identifier, `@<payload-file>` syntax, and optional deadline.

## Behavior

### Synopsis

```text
synchestra runner invoke @<payload-file> --runner <id> --handler <name> --invocation-id <id> [--deadline <rfc3339>] [--format text|json]
```

### Parameters

| Parameter | Required | Description |
|---|---|---|
| [`@<payload-file>`](_args/payload-file.md) | Yes | Path to the JSON file whose exact bytes are delivered. |
| [`--runner`](_args/runner.md) | Yes | Exact registered runner that must execute the handler. |
| [`--handler`](_args/handler.md) | Yes | One of the closed WB handler names. |
| [`--invocation-id`](_args/invocation-id.md) | Yes | Caller-owned request identity; for `wb.session.accept.v1`, this is the WB handoff ID. |
| [`--deadline`](_args/deadline.md) | No | Immutable RFC3339 deadline metadata. |
| [`--format`](../../_args/format.md) | No | Output format: `text` (default) or `json`. |

### Payload loading and ownership

The CLI reads at most 1 MiB plus one sentinel byte and rejects empty,
oversized, or syntactically invalid JSON before contacting the Hub. It retains
the exact file bytes, including whitespace, as an opaque payload; only byte
count and SHA-256 digest are derived. It does not read a handoff ID, executable,
argv, shell expression, repository target, or session identity from the JSON.

The current Git checkout supplies repository identity and the immutable `HEAD`
revision. Resolution uses read-only Git operations and does not stage, commit,
switch branches, create state, or modify tracked or untracked caller files.

### Durable dispatch adapter

The typed invocation is encoded through the dispatch-v1 compatibility adapter
and submitted with the existing dispatch client. Handler routing uses only the
closed handler registry and its synthetic scheduler capability; request payload
data cannot affect routing. The durable `Dispatch.created_at` is the canonical
invocation creation time, so the opaque invocation envelope has no
caller-generated creation timestamp.

For `wb.session.accept.v1`, `--invocation-id` is passed to the bounded
`WBHandoffIdempotencyKey` derivation. Repeating the same handoff ID with the
same immutable request returns the original dispatch; the CLI reads its
existing attempts through the status endpoint so a terminal receipt artifact
is returned without creating a retry attempt. Reusing that ID with a different
payload digest conflicts at dispatch creation. An explicitly supplied deadline
is part of the immutable request and must match on replay.

### Lifecycle observation and output

The invocation uses the existing operations:

```text
synchestra runner dispatch status <dispatch-id>
synchestra runner dispatch logs <dispatch-id>
synchestra runner dispatch retry <dispatch-id>
synchestra runner dispatch cancel <dispatch-id>
```

No invocation-specific queue or attempt type exists. Status exposes the same
attempt state, worker identity, lease generation, timestamps, cancellation,
structured failure code/stage/retryability and log reference, and terminal
artifact references used by ordinary dispatch work. The logs operation first
reads status through the existing endpoint to identify the reserved invocation
envelope, then omits message text from its invocation-safe event projection.
Retry appends an attempt and cancel records durable intent through the existing
Hub routes and fencing rules. Ordinary dispatch log and lifecycle output is
unchanged.

Text and JSON output expose typed invocation metadata: protocol version,
invocation ID, handler, payload digest and size, canonical dispatch creation
time, optional deadline, runner, repository evidence, dispatch state, and safe
attempt fields. They never serialize the compatibility `project_context`, raw
payload, synthetic handler agent/model selectors, compatibility branch
coordinates, terminal result summaries, terminal failure messages,
cancellation reasons, or invocation log messages. Artifact and log references
remain available for durable observation without turning lifecycle output into
a payload channel.

With `--format json`, successful creation uses this stable top-level shape:

```json
{
  "resolved": {
    "operation": "invoke",
    "repository": {},
    "runner": "personal-vm",
    "invocation": {
      "protocol_version": "synchestra.handler-invocation.v1",
      "id": "handoff-42",
      "handler": "wb.session.accept.v1",
      "payload_digest": "sha256:…",
      "payload_size": 123,
      "created_at": "2026-08-25T12:00:00Z"
    }
  },
  "dispatch": {},
  "attempts": [],
  "created": true
}
```

Observation and mutation commands retain their existing top-level
`resolved`/`dispatch`/`attempts` or `attempt` shapes. Typed projection is
conditional: ordinary dispatch JSON remains compatible with the existing
contract.

### Exit codes

The command uses the shared runner-dispatch exit codes. Invalid flags, payload
syntax, handler names, invocation IDs, JSON, or deadlines return `2`; an
idempotency conflict returns `1`; missing repository context returns `3`;
authentication and Hub/runner errors retain their existing stable codes.

## Dependencies

- [cli/runner](../README.md) — parent command group
- [cli/runner/dispatch](../dispatch/README.md) — durable create and lifecycle operations
- [dispatch](../../../dispatch/README.md) — queue, attempts, leases, logs, retry, and cancellation
- [wb-session-transport](../../../wb-session-transport/README.md) — typed WB handler and ownership boundary
- [repo-config](../../../repo-config/README.md) — repository identity and project context

## Acceptance Criteria

1. A valid `@<payload-file>` for either registered WB handler creates a normal durable dispatch with exact runner and capability constraints, immutable repository evidence, and payload digest/size metadata.
2. Invalid, empty, or oversized JSON and unknown handler names fail before any Hub mutation and do not echo payload bytes.
3. Repeating an accepted WB handoff invocation returns its original terminal attempt and artifact reference, while changing the payload under the same handoff ID returns a conflict.
4. Existing status, logs, retry, and cancel routes operate on the invocation's original dispatch and expose no raw payload or compatibility-envelope fields.
5. Invocation creation leaves the caller's branch, index, tracked changes, and untracked files unchanged.

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
