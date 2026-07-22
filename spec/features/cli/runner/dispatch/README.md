# Feature: CLI Runner Dispatch

**Status:** In Progress

## Summary

Creates and observes durable remote work through the frozen `synchestra.dispatch.v1` contract. Creation accepts an ad-hoc prompt or a SpecScore Plan/Task target, returns a durable dispatch ID immediately, and never waits for a worker session.

## Synopsis

```text
synchestra runner dispatch [<target>] [--prompt <text> | --plan <target> | --task <target>]
  [--runner <id>] [--profile fast|balanced|large] [--agent <adapter>]
  [--model <selector>] [--effort <value>] [--format text|json]

synchestra runner dispatch status <dispatch-id> [--format text|json]
synchestra runner dispatch logs <dispatch-id> [--cursor <n>] [--format text|json]
synchestra runner dispatch retry <dispatch-id> [--reason <text>] [--format text|json]
synchestra runner dispatch cancel <dispatch-id> [--reason <text>] [--format text|json]
```

Exactly one creation source is required. The positional target searches both kinds; `--plan` and `--task` constrain resolution to one SpecScore kind. All source forms are mutually exclusive.

Arguments are documented in [_args/](./_args/README.md). Creation and observation use the global [`--format`](../../_args/format.md) semantics.

## Deterministic repository resolution

Creation resolves the containing Git root, credential-free `origin` URL, canonical repository ID, current full `HEAD^{commit}` object ID, symbolic current branch as audit-only `base_ref`, nearest project root, optional Hub project ID, and repository-relative project subdirectory. Hosted SSH-style origins are normalized to credential-free HTTP(S) clone URLs; the CLI does not perform an SSH fallback.

Only read-only Git commands are used. The branch, `HEAD`, refs, index, tracked worktree, staged content, and untracked files remain unchanged, including in a dirty checkout. The immutable base revision—not `base_ref`—is execution authority.

## Source resolution

An ad-hoc source records the prompt directly and does not create a Synchestra Task.

SpecScore candidates are read from committed `HEAD`, not dirty working-tree bytes. Resolution proceeds deterministically:

1. committed path;
2. exact resource ID or canonical alias;
3. normalized exact name;
4. normalized substring name.

Zero matches return not-found. Multiple matches return invalid-arguments with sorted candidates. The CLI never prompts. A resolved target records kind, canonical ID, repository-relative path, the immutable target revision, and a SHA-256 content snapshot hash. Numbered Task sections inside Plan documents are addressable as `{plan-id}#task-{number}`.

## Execution selectors

The provider-neutral profiles are `fast`, `balanced`, and `large`; the default requested profile is `balanced`. `--agent`, `--model`, and `--effort` are passed unchanged. Model values may be exact names or adapter selectors such as `sonnet`. The CLI does not map models or encode fallback instructions in prompt prose. It records the v1 exact-selector default, `fallback.mode: reject`; routing and mappings belong to the Hub scheduler.

`--runner` sets the hard `worker_constraints.runner_id`. Omitting it permits any authorized eligible worker.

## Hub transport and authentication

The CLI sends Bearer-authenticated requests to these public caller endpoints:

| Operation | Method and route |
|---|---|
| create | `POST /v1/dispatches` |
| status | `GET /v1/dispatches/{id}` |
| logs | `GET /v1/dispatches/{id}/logs?cursor={n}` |
| retry | `POST /v1/dispatches/{id}/retry` |
| cancel | `POST /v1/dispatches/{id}/cancel` |

It never calls scheduler/worker claim or attempt-owner endpoints. Each mutation carries a generated idempotency/operation ID. `created_by` and `requested_by` record configurable client provenance (`synchestra-cli` by default); Bearer authentication remains authoritative.

Configuration follows existing CLI precedence. `SYNCHESTRA_URL`, `SYNCHESTRA_TOKEN`, and `SYNCHESTRA_ACTOR` override project/global configuration. A project's `synchestra.yaml` `hub.endpoint` overrides the global endpoint. Global values are read from `SYNCHESTRA_CONFIG`, `~/.synchestra.yaml`, or the compatibility path `~/.synchestra/config.yaml`. Tokens are never printed.

Every response must carry exactly `synchestra.dispatch.v1`, including returned Dispatch and Attempt records. A mismatch exits `82` with an upgrade-the-older-component diagnostic; the CLI never guesses or downgrades.

## Output

Text creation output has stable `Resolved` and `Dispatch` blocks. It includes source identity, repository/revision, requested selectors, runner constraint, durable dispatch ID, status, and creation time. It does not echo the prompt.

JSON output is one object. Success contains `resolved` and the operation payload (`dispatch`, `attempts`, or `logs`); failure contains `resolved` and `error`. A success object never contains `error`, and a failure object never contains `dispatch`.

## Exit codes

| Code | Meaning |
|---:|---|
| `0` | Success |
| `1` | Conflict |
| `2` | Invalid/unsupported arguments, selector, or ambiguous target |
| `3` | Dispatch or target not found |
| `4` | Invalid dispatch lifecycle state |
| `10` | Unexpected response/runtime failure |
| `80` | Explicit runner not found |
| `81` | No eligible worker satisfies the constraints |
| `82` | Incompatible dispatch protocol |
| `101` | Unauthenticated |
| `102` | Hub unreachable |

## Acceptance Criteria

1. Ad-hoc creation emits the v1 request shape and does not create a Task.
2. Plan and Task path/ID/name resolution records an immutable revision and content hash; ambiguity fails before any Hub request.
3. Creation leaves branch, `HEAD`, refs, index, staged/unstaged content, and untracked files unchanged in a dirty checkout.
4. Optional runner/profile/agent/model/effort fields are passed as requested without CLI routing or prompt fallback prose.
5. Create returns a durable dispatch ID immediately; status/logs/retry/cancel route only to the public caller endpoints above.
6. Text output is stable and JSON is exactly one object with `resolved` plus a success payload or error.
7. Invalid arguments, authentication, runner matching, lifecycle errors, transport failure, and protocol mismatch use the documented stable exit codes.

## Outstanding Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
