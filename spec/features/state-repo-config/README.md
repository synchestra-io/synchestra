# Feature: State Repo Config

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-repo-config?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-repo-config?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-repo-config?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-repo-config?op=request-change) |

**Status:** Approved

**Source Ideas:** unified-project-definition

## Summary

Defines `synchestra-state.yaml`, the self-identifier file that lives at the root of every Synchestra **state location** — whether that is a dedicated state repository, an orphan branch in the project's main repo (embedded mode), or a Hub-managed state worktree. The file declares which spec repos this state belongs to, distinguishing legitimate state locations from arbitrary directories that happen to look like one.

State locations have an independent lifecycle from the SpecScore spec layer: they are not specscore-managed, do not contain a `spec/` tree, and do not have their own `specscore.yaml`. The state-repo-config file is what tells tools "this directory is a Synchestra state location."

## Problem

When a user has a state location on disk (a checked-out separate-repo state, an orphan-branch worktree at `.synchestra/`, or a Hub-managed clone), tools need a deterministic way to (a) confirm it is a Synchestra state location and (b) discover which spec repo(s) point at it. Without a self-identifier file, this requires brittle path inference and side-channel state.

The legacy `synchestra-state-repo.yaml` filename came from the three-file scheme this Idea is collapsing. Renaming to `synchestra-state.yaml` aligns with the new two-tier naming (`synchestra.yaml` for project root, `synchestra-state.yaml` for state location) and removes the redundant `-repo` suffix that was a vestige of the spec/state/code triad.

## Behavior

### File name and location

#### REQ: file-name

Every Synchestra state location MUST contain a file named exactly `synchestra-state.yaml` at the location's root. For embedded-state mode, the location root is the orphan branch's worktree root (e.g., `.synchestra/` in the working tree). For separate-repo mode, it is the state repo's root.

#### REQ: schema-header-comment

Line 1 of `synchestra-state.yaml` MUST be exactly:

```yaml
# Synchestra State Schema: https://synchestra.md/state-repo-config
```

Any other line-1 content (blank line, different comment, document marker, BOM) is invalid and tools MUST emit a hard error.

### Spec-repo back-references

State locations are owned by one or more spec repos. The back-reference is what permits tools to navigate from state to specs (e.g., to look up a task's parent feature definition).

#### REQ: spec-repos-list

`spec_repos:` is a REQUIRED non-empty list of spec-repo URLs. Each entry is the canonical origin URL of a spec repo whose `specscore.yaml` (and Synchestra `synchestra.yaml`) declare a `state:` block pointing at this state location. A state-repo file with no `spec_repos:` field, or with an empty list, is a hard error — orphan state locations are not supported.

### Mode

#### REQ: mode-field

`synchestra-state.yaml` MUST declare its mode via a top-level `mode:` field, one of `embedded`, `separate-repo`, or `hub-managed`. The field is required because the three modes have distinct lifecycles and tools must know which to apply; layout-based inference would silently mis-classify edge cases (orphan-branch-without-worktree, bare-clone tooling paths, hub-managed checkouts at non-standard paths). Omitting `mode:` is a hard error. The value MUST be consistent with the spec repo's `synchestra.yaml#state.mode`; tools MAY surface an advisory note (per the peer `repo-config` Feature's use of the term — a non-blocking diagnostic written to stderr that does NOT alter exit code) on inconsistency, but MUST NOT silently rewrite either file.

### Title

#### REQ: title-field

`synchestra-state.yaml` MAY declare an optional `title:` for human-readable display. When omitted, tools MUST use the spec repo's `project.title` from `specscore.yaml` as the displayed name.

### Unknown fields

#### REQ: unknown-fields-preserved

Tools MUST ignore unknown fields at the root and round-trip them unchanged. No warnings or errors for unknown fields.

### Example

A typical `synchestra-state.yaml` for an embedded state location:

```yaml
# Synchestra State Schema: https://synchestra.md/state-repo-config

mode: embedded
title: Acme Service
spec_repos:
  - https://github.com/acme/service
```

A multi-spec-repo state location (rare; advanced use):

```yaml
# Synchestra State Schema: https://synchestra.md/state-repo-config

mode: separate-repo
spec_repos:
  - https://github.com/acme/service
  - https://github.com/acme/service-frontend
```

## Architecture

Parsing follows the same pattern as `synchestra.yaml` and `specscore.yaml`: a small Go struct with a custom YAML unmarshaller, line-1 header validation, and a `Validate()` method enforcing in-memory invariants. The parser package is colocated with the `repo-config` parser; both are part of the synchestra config-files layer.

## Acceptance Criteria

### AC: state-file-happy-path

**Requirements:** state-repo-config#req:file-name, state-repo-config#req:schema-header-comment, state-repo-config#req:spec-repos-list, state-repo-config#req:mode-field

**Given** a directory containing a file named exactly `synchestra-state.yaml` whose first line is the canonical schema-header comment, whose body declares `mode:` (one of the three enum values) and a non-empty `spec_repos:` list with at least one URL
**When** Synchestra tooling loads the file
**Then** the load succeeds and the parsed `mode`, `spec_repos`, and any optional `title` are exposed; round-tripping the parsed result back to disk produces the same bytes as the input.

### AC: state-file-name-and-header-errors

**Requirements:** state-repo-config#req:file-name, state-repo-config#req:schema-header-comment

**Given** a directory containing one of: (a) a file named `synchestra-state-repo.yaml` (legacy name) at the location root; (b) a file named `synchestra-state.yaml` whose first line is blank, a different comment, or a YAML document marker
**When** tools attempt to load the state location
**Then** for (a) the load fails with a hard error naming the legacy filename and citing `state-repo-config#req:file-name`; for (b) the load fails with a hard error citing `state-repo-config#req:schema-header-comment`.

### AC: state-file-content-errors

**Requirements:** state-repo-config#req:spec-repos-list, state-repo-config#req:mode-field

**Given** a `synchestra-state.yaml` with a valid header that nonetheless contains one of: (a) no `spec_repos:` field; (b) `spec_repos: []` (empty list); (c) no `mode:` field; (d) `mode: helm-managed` (unknown value)
**When** tools load the file
**Then** for (a) and (b) the load fails citing `state-repo-config#req:spec-repos-list`; for (c) the load fails citing `state-repo-config#req:mode-field` with text "mode: is required"; for (d) the load fails citing `state-repo-config#req:mode-field` with text naming the unknown value and listing the accepted enum values; in every case no implicit default is silently applied.

### AC: mode-consistency

**Requirements:** state-repo-config#req:mode-field

**Given** a `synchestra-state.yaml` declaring `mode:` and a sibling-discovered `synchestra.yaml` (in the spec repo) declaring `state.mode:`
**When** tools resolve the effective state mode
**Then** when both values agree, no diagnostic is emitted; when they disagree, an advisory note (a non-blocking message written to stderr that does NOT alter exit code, per the peer `repo-config` Feature's terminology) is surfaced naming both files and both values; tools MUST NOT silently rewrite either file to reconcile.

### AC: extensions-preserved

**Requirements:** state-repo-config#req:unknown-fields-preserved

**Given** a `synchestra-state.yaml` with unknown fields at the root
**When** tools load and re-write the file
**Then** every unknown field round-trips byte-identical; no warning or error is emitted.

## Open Questions

- Whether `synchestra-state.yaml` should also carry a `title` field that wins over `specscore.yaml#project.title` is open. Current spec defers to specscore; could change.
- The `https://synchestra.md/state-repo-config` schema-pointer URL is a forward reference until synchestra.md publishes the schema page.
- Hub-managed state locations may need additional metadata (Hub URL, project ID, last sync timestamp). Deferred to the future hub-managed-state Feature.

---
*This document follows the https://specscore.md/feature-specification*
