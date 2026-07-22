---
format: https://specscore.md/feature-specification
status: Approved
---

# Feature: Dispatch Agent Command

**Status:** Approved
**Source Ideas:** [Remote Dispatch](https://github.com/synchestra-io/synchestra-marketing/blob/main/ideas/remote-dispatch.md)

## Summary

The Synchestra AI plugin provides a concise, human-visible `/dispatch` command that accepts conversational work plus an optional agent/model selector and delegates to the deterministic Synchestra dispatch CLI. Queueing, repository resolution, retries, and status are implemented by Synchestra, not by prompt instructions.

## Problem

Remote dispatch must be as easy to invoke as ordinary agent work while remaining reliable and cross-platform. A large skill that reproduces scheduler logic is non-deterministic and expensive; hiding the operation only under a low-level runner resource makes the primary user value hard to discover.

## Behavior

### Invocation

Canonical examples include:

```text
/dispatch @sonnet Update dal-go deps to latest
/dispatch @fast Fix the failing Firestore test
/dispatch status dsp_01...
/dispatch logs dsp_01...
/dispatch cancel dsp_01...
```

The plugin may namespace the command as `/synchestra:dispatch` where required and provide `/dispatch` as a collision-safe alias where supported.

### Skill structure

- `commands/dispatch.md` is the human slash-command entry point for Claude Code.
- The resource-level `runner` skill contains `references/dispatch.md` and remains the canonical progressive-disclosure wrapper for `synchestra runner dispatch` or its finalized CLI spelling.
- Other platforms expose equivalent skill metadata without duplicating operational behavior.
- The skill validates that the CLI is present, resolves only conversational selector syntax, calls the CLI, and surfaces its structured result or error verbatim.

### Selector mapping

The plugin recognizes provider-neutral `@fast`, `@balanced`, and `@large`, plus adapter-specific aliases such as `@haiku`, `@sonnet`, and `@opus`. It sends the selector to the CLI; it does not hard-code concrete model IDs or silently choose a fallback.

## Acceptance Criteria

### AC: command-submits-ad-hoc-work

**Given** the user invokes `/dispatch @sonnet <prompt>` inside a Git repository
**When** the command runs
**Then** it calls the deterministic Synchestra CLI with the resolved prompt and selector and returns the dispatch ID, status, repository, and requested profile/model.

### AC: command-does-not-execute-locally

**Given** a successful dispatch submission
**When** the command returns
**Then** it has not edited the repository or run the implementation task in the caller's agent session.

### AC: structured-errors-surfaced

**Given** authentication, repository resolution, selector validation, or dispatch creation fails
**When** the CLI returns a documented non-zero result
**Then** the command surfaces the actionable error and performs no improvised queue or SSH fallback.

### AC: observation-verbs-delegate

**Given** a dispatch ID
**When** the user requests status, logs, retry, or cancellation
**Then** the command delegates to the corresponding deterministic CLI operation and preserves its exit semantics.

### AC: specstudio-not-required

**Given** the Synchestra plugin and CLI are installed but SpecStudio is absent
**When** the user invokes `/dispatch`
**Then** the complete command workflow remains available.

## Dependencies

- [agent-skills](../README.md) — plugin structure and progressive disclosure
- [dispatch](../../dispatch/README.md) — product lifecycle and result contract
- [cli/runner/dispatch](../../cli/runner/dispatch/README.md) — deterministic CLI operation
- [ADR-0006](../../../decisions/0006-queued-remote-dispatch-boundary.md) — ownership boundary

## Outstanding Questions

1. Whether `/dispatch` is a universal unnamespaced alias on every supported agent platform depends on each platform's collision rules; the namespaced command is always valid.

## Implementation Evidence

[`ai-plugin-synchestra` PR #2](https://github.com/synchestra-io/ai-plugin-synchestra/pull/2), through commit `42febf9ac51057020284419cbedd471cfaece5a1`, adds the namespaced `/synchestra:dispatch` command and thin runner reference. It parses `@fast`, `@balanced`, `@large`, `@haiku`, `@sonnet`, and `@opus`, and routes create/status/logs/retry/cancel to the deterministic CLI without embedding scheduler, repository, SSH, retry, or model-mapping logic in prompt prose. Structured CLI failures are preserved without silent retry, selector substitution, protocol fallback, credential inspection, or SSH bypass. Claude plugin manifest validation and semantic command tests pass. An unnamespaced alias remains platform/collision dependent as specified.

---
*This document follows the https://specscore.md/feature-specification*
