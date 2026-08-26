# Feature: Synchestra Repo Config

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/repo-config?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/repo-config?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/repo-config?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/repo-config?op=request-change) |

**Status:** Approved

**Source Ideas:** unified-project-definition

## Summary

Defines `synchestra.yaml`, the single repo-level config file that holds Synchestra-only orchestration metadata. The file lives at the repository root next to `specscore.yaml`. Project identity (title, host, org, repo, repositories) is **read** from `specscore.yaml` — never duplicated. `synchestra.yaml` holds only what SpecScore does not: state-repo location, sync policy, hub registration, runner pinning, and similar orchestration concerns. The schema mirrors SpecScore's `repo-config` pattern: a fixed line-1 schema-pointer comment, a single authoritative root file, no front-matter, an empty file is valid.

## Problem

Synchestra today carries a three-file split (`synchestra-spec-repo.yaml`, `synchestra-code-repo.yaml`, `synchestra-state-repo.yaml`) that predates SpecScore's unified `specscore.yaml`. Two of those files duplicate fields SpecScore already owns (`title`, `repositories`); maintaining them parallel to specscore.yaml creates real risk of drift and a confusing mental model. Collapsing into one synchestra-owned file plus reads from specscore.yaml yields a clean two-file outcome at a project root: SpecScore owns project identity, Synchestra owns orchestration.

Embedding synchestra fields as extension keys inside `specscore.yaml` was considered and rejected (see source Idea Alternatives) because it re-mixes the two layers and creates a flat-namespace collision risk.

## Behavior

### File name and location

#### REQ: file-name

Every Synchestra-managed project MUST contain a file named exactly `synchestra.yaml` at the repository root. Any other name (including `.synchestra.yaml`, `synchestra-config.yaml`, or location below the root) is invalid.

#### REQ: schema-header-comment

Line 1 of `synchestra.yaml` MUST be exactly:

```yaml
# Synchestra Repo Config Schema: https://synchestra.md/repo-config
```

A file whose line 1 is anything else — blank line, a different comment, a YAML document marker (`---`), or a leading BOM followed by other content — is invalid. Tools MUST emit a hard error.

#### REQ: empty-config-valid

A `synchestra.yaml` containing only the schema-header comment (with an optional trailing newline) is valid. All field defaults defined by this Feature apply.

### Composition with specscore.yaml

`synchestra.yaml` and `specscore.yaml` together describe a Synchestra-managed project. The boundary is sharp.

#### REQ: specscore-yaml-as-identity-source

Synchestra tooling MUST read project identity (`project.title`, `project.host`, `project.org`, `project.repo`, `project.repositories`) from `specscore.yaml` at the same repository root. Synchestra MUST NOT duplicate any of those fields inside `synchestra.yaml`. The following are each a hard error:

- A top-level `project:` key in `synchestra.yaml` (the entire identity block belongs in `specscore.yaml`).
- Any of the bare top-level keys `title:`, `host:`, `org:`, `repo:`, or `repositories:` in `synchestra.yaml`.

The error message MUST name the offending key and direct the user to `specscore.yaml` as the canonical home.

#### REQ: missing-specscore-yaml-allowed

A repository MAY have a `synchestra.yaml` without a sibling `specscore.yaml`. In that case Synchestra tools fall back to inferring project identity from the working directory and `origin` git remote — the same defaults SpecScore would have applied if `specscore.yaml` were present and empty. Tools MAY surface an advisory note recommending that the user run `specscore init` for a complete project definition; they MUST NOT fail on the missing file alone.

### State configuration

#### REQ: state-block

`synchestra.yaml` MAY contain a top-level `state:` block describing where Synchestra state for this project lives. The block's accepted fields are mode-specific. The full `(field × mode)` matrix is:

| Field | `embedded` | `separate-repo` | `hub-managed` |
|---|---|---|---|
| `mode` | required | required | required |
| `repo` | forbidden | required | forbidden |
| `branch` | optional (default `synchestra-state`) | forbidden | forbidden |
| `sync` | optional (see `sync-policy-block`) | optional | optional |

In `hub-managed` mode the `state:` block carries only `mode:` (and optionally `sync:`); the Hub identity that locates the state lives in the top-level `hub:` block (see `hub-block`). Any field whose cell is "forbidden" in the chosen mode MUST cause a hard error citing this REQ, naming both the offending field and the mode.

When the entire `state:` block is omitted, tools MUST treat the project as having no Synchestra state configured. Operations that require state (task management, embedded-state read/write) fail with a hard error pointing the user at `synchestra init`.

#### REQ: state-mode-enum

`state.mode` MUST be one of `embedded`, `separate-repo`, or `hub-managed`. Unknown values are a hard error. The enum is closed in v1.

#### REQ: sync-policy-block

`state.sync` is an OPTIONAL mapping with `pull` and `push` fields, each one of `on_commit`, `on_session_end`, `on_interval`, or `manual`. When `state.sync` is omitted, tools MUST default to `pull: on_commit, push: on_commit`. When present, both fields MUST be specified — partial mappings are a hard error.

### Hub registration

#### REQ: hub-block

`synchestra.yaml` MAY contain a top-level `hub:` block declaring the project's Hub identity. The block carries an `id` field (the canonical Hub project identifier) and optional `endpoint`. When the block is omitted, tools MUST behave as if the project is not Hub-registered and proceed in local-only mode.

### Unknown fields

#### REQ: unknown-fields-preserved

Tools MUST ignore unknown fields at the root, inside `state:`, inside `state.sync:`, and inside `hub:`. Unknown fields MUST round-trip unchanged on read/write and MUST NOT cause a validation warning or error. This permits future Synchestra Features (and adjacent orchestration tools) to extend the schema without invalidating existing files.

### Example

A maximal `synchestra.yaml` exercising every documented field:

```yaml
# Synchestra Repo Config Schema: https://synchestra.md/repo-config

state:
  mode: separate-repo
  repo: https://github.com/acme/service-state
  sync:
    pull: on_commit
    push: on_session_end

hub:
  id: acme/service
  endpoint: https://api.synchestra.io/
```

A minimal `synchestra.yaml` for an embedded-state project:

```yaml
# Synchestra Repo Config Schema: https://synchestra.md/repo-config

state:
  mode: embedded
```

The empty-but-valid form (no state, no hub — typically a transitional state):

```yaml
# Synchestra Repo Config Schema: https://synchestra.md/repo-config
```

## Architecture

The `synchestra.yaml` parser is a small Go package — a peer of SpecScore's `projectdef` — that exposes:

- A `SynchestraConfig` struct with `State *StateConfig`, `Hub *HubConfig`, and `Extras map[string]any`.
- `Read(dir) (SynchestraConfig, error)` and `Write(dir, cfg) error`.
- `ValidateSchemaHeader([]byte) error` for line-1 enforcement.
- `Validate() error` for in-memory invariants (mode-specific field requirements, sync-policy completeness, identity-field rejection).

The parser does NOT read `specscore.yaml`; consumers compose the two reads themselves. This keeps the dependency direction clean and makes synchestra.yaml independently parseable.

## Acceptance Criteria

### AC: file-recognized

**Requirements:** repo-config#req:file-name, repo-config#req:schema-header-comment, repo-config#req:empty-config-valid

**Given** a repository with a file named exactly `synchestra.yaml` at its root whose first line is the canonical schema-header comment
**When** Synchestra tooling loads the file
**Then** the load succeeds; an empty file containing only the header is valid; a file with any other name, a malformed/missing header, or a header on any line other than line 1 produces a hard error naming the violated REQ.

### AC: identity-not-duplicated

**Requirements:** repo-config#req:specscore-yaml-as-identity-source

**Given** a `synchestra.yaml` that declares any of (a) a top-level `project:` block, or (b) a bare top-level `title:`, `host:`, `org:`, `repo:`, or `repositories:` field
**When** Synchestra tooling loads the file
**Then** for every forbidden key shape the load fails with a hard error naming the specific offending field and citing `repo-config#req:specscore-yaml-as-identity-source`; the user is directed to put project identity in `specscore.yaml`.

### AC: composes-with-specscore

**Requirements:** repo-config#req:specscore-yaml-as-identity-source, repo-config#req:missing-specscore-yaml-allowed

**Given** a repository root with a valid `synchestra.yaml` and either (a) a sibling `specscore.yaml` or (b) no `specscore.yaml`
**When** Synchestra tooling resolves the project's effective title, host, org, repo, and repositories
**Then** in case (a) the values come from `specscore.yaml` (with SpecScore's own defaults applied for omitted fields); in case (b) the values come from the working directory basename and `origin` git remote (the SpecScore-empty defaults), and an advisory recommending `specscore init` MAY be surfaced but MUST NOT be a hard error.

### AC: state-roundtrip

**Requirements:** repo-config#req:state-block, repo-config#req:state-mode-enum, repo-config#req:sync-policy-block

**Given** a `synchestra.yaml` whose `state:` block uses each documented mode (`embedded`, `separate-repo`, `hub-managed`) with mode-appropriate fields and a complete `sync:` policy
**When** tools load the config and round-trip it back to disk
**Then** the on-disk bytes are identical for each well-formed entry; mode-specific field requirements are enforced per the `(field × mode)` matrix in `state-block` (e.g., `state.repo` required when `mode: separate-repo` and forbidden in `embedded` and `hub-managed`; `state.branch` permitted only in `embedded`); a `sync:` mapping missing either `pull` or `push` produces a hard error citing `repo-config#req:sync-policy-block`; an unknown `mode` value produces a hard error citing `repo-config#req:state-mode-enum`.

### AC: sync-defaults-applied-when-omitted

**Requirements:** repo-config#req:sync-policy-block

**Given** a `synchestra.yaml` whose `state:` block declares a `mode:` but no `sync:` field
**When** tools load the config
**Then** the loaded representation exposes the effective sync policy as `pull: on_commit, push: on_commit`; on round-trip back to disk the `sync:` block is NOT materialized (the file remains byte-identical to the input).

### AC: hub-block-roundtrip

**Requirements:** repo-config#req:hub-block

**Given** a `synchestra.yaml` declaring a `hub:` block with `id` and optional `endpoint`, and another `synchestra.yaml` with no `hub:` block
**When** tools load both configs
**Then** the first exposes the parsed `hub.id` (and `hub.endpoint` when present) and round-trips byte-identical; the second exposes a nil `hub` field and tools treat the project as not Hub-registered (local-only mode); neither case emits a warning or error.

### AC: missing-state-fails-with-guidance

**Requirements:** repo-config#req:state-block

**Given** a `synchestra.yaml` containing only the schema-header comment (no `state:` block)
**When** tools that require state — for example, a hypothetical `synchestra task new` — are invoked against this project
**Then** the operation fails with a hard error whose message names the missing `state:` block and explicitly directs the user to run `synchestra init`; tools that do NOT require state (e.g., `synchestra info`) succeed without error.

### AC: extensions-preserved

**Requirements:** repo-config#req:unknown-fields-preserved

**Given** a `synchestra.yaml` with unknown fields at the root, inside `state:`, inside `state.sync:`, and inside `hub:`
**When** tools load and re-write the file
**Then** every unknown field round-trips byte-identical; no warning or error is emitted.

## Open Questions

- The schema-pointer URL `https://synchestra.md/repo-config` mirrors SpecScore's URL convention. The synchestra.md domain may not yet host the schema page; the URL is a forward reference. Resolve when synchestra.md publishes its schema documentation.
- Whether `sync:` policy values should be a tighter enum (e.g. add `never`) is deferred to the sync-policy Feature when it is specified.
- Whether `hub:` should also support an authenticated-token reference field is deferred to the host-auth / hub-registration Feature.
- The interaction between `state.mode: embedded` here and the existing [embedded-state](../embedded-state/README.md) Feature needs reconciliation: this Feature describes the configuration shape; embedded-state describes the orphan-branch mechanics. A future revision should cross-link the two.

---
*This document follows the https://specscore.md/feature-specification*
