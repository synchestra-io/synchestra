# Feature: CLI Runner

**Status:** In Progress

## Summary

Command group under the `synchestra` CLI for managing remote runners and dispatching work to them. This feature is the CLI surface for the product-level [Runner](../../runner/README.md) feature — it does not redefine what a runner is; it defines how a user interacts with runners from the terminal.

## Problem

Users and AI agents need a programmatic surface to:

1. Register runners they own (VMs, Cloud Run services) with the Synchestra Hub.
2. List runners available to them.
3. Send an ad-hoc prompt, Plan, or Task for remote execution.
4. Inspect a runner's health without opening the web UI.

The existing [`task`](../task/README.md) command group handles local task lifecycle; it has no concept of "somewhere else." Without a dedicated runner command group, remote execution either leaks into `task` (diluting its scope) or is restricted to the UI (breaking CLI-first workflows).

## Behavior

### Command group structure

Commands follow the CLI's standard `synchestra <group> <verb>` pattern. The group is `runner`; verbs operate on the runner resource (management) or dispatch work through it.

```
synchestra runner <verb>
```

Each verb is its own subfeature with a dedicated spec directory under this one.

### Runner identity

Runners are identified by user-chosen names, unique per authenticated user (e.g., `hetzner-vm1`, `gcp-cloudrun-dev`). The CLI is indifferent to a runner's underlying runtime — VM, Cloud Run, or future runtimes — because runtime selection is a hub and [Runner](../../runner/README.md) concern, not a CLI concern.

Optional metadata attached at `add` time (`--meta key=value`) is displayed in `list` output for the user's own recall. It does not affect dispatch routing in MVP.

### Authentication

All verbs except trivial help invocations require the calling user to be authenticated via [`synchestra auth login`](../auth/README.md). Unauthenticated calls fail with the standard unauthenticated exit code defined by the [CLI Auth](../auth/README.md) feature.

### Local checkout behavior

`runner dispatch` resolves the caller's current immutable Git revision and submits a durable Hub record. It does not check out a branch, stage files, update the index, create a Task for ad-hoc work, or otherwise modify the caller repository. Dispatch observation and control operations use the public Hub API only.

## Contents

| Directory | Description |
|---|---|
| [dispatch/](dispatch/README.md) | Dispatch a plan or task to a runner, creating a session |

### dispatch

Accepts an ad-hoc prompt or an unambiguous SpecScore Plan/Task target and returns a durable dispatch ID immediately. The `status`, `logs`, `retry`, and `cancel` operations observe or control that dispatch through public Hub endpoints.

### Planned subfeatures

These verbs are part of this feature's intended surface but do not yet have their own spec directories:

| Verb | Description |
|---|---|
| `add` | Register a runner with the Hub under a user-chosen name |
| `remove` | Unregister a runner |
| `list` | List runners available to the authenticated user |
| `status` | Report a runner's health and active session count |

## Dependencies

- [cli](../README.md) — parent feature; defines command conventions, sync flags, and exit code contract
- [runner](../../runner/README.md) — product-level definition of what a runner is
- [host-auth](../../host-auth/README.md) — registered runners authenticate to the Hub via host-auth; the CLI consumes runner identities that pass host-auth
- [cli/auth](../auth/README.md) — user-to-Hub authentication required by all runner verbs
- [cli/session](../session/README.md) — `dispatch` creates a session; session verbs observe and control it

## Acceptance Criteria

1. Every Hub operation under `synchestra runner` fails with an "unauthenticated" exit code when the caller has no valid user credentials.
2. Runner names are unique per authenticated user; `add` with a duplicate name fails rather than overwriting.
3. The CLI exposes no user-visible distinction between VM and Cloud Run runners beyond optional metadata.
4. Dispatch operations leave the caller's checkout, index, refs, and files unchanged.

## Outstanding Questions

1. Should `runner list` include metadata fields in default output, or require a flag (`--long`, `--format json`) to surface them?
2. Does `runner status <name>` need a dedicated subfeature spec, or can its behavior be fully captured in this group README?
3. When the authenticated user has zero registered runners, should a future runner-management command provide registration guidance in addition to the stable dispatch error?

---
*This document follows the https://specscore.md/feature-specification*
