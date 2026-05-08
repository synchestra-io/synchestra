# Idea: Unified Project Definition

**Status:** Approved
**Date:** 2026-05-08
**Owner:** alexander.trakhimenok
**Promotes To:** —
**Supersedes:** —
**Related Ideas:** —

## Problem Statement

How might we collapse Synchestra's three-file project config into a single `synchestra.yaml` file that reads project identity from `specscore.yaml` (with multi-role repository entries upstream in SpecScore), and reshape `synchestra init` around the unified two-file model?

## Context

While ideating on `synchestra init` (and what it would do beyond the existing `synchestra project init`), we found that `synchestra-spec-repo.yaml` duplicates fields already canonical in `specscore.yaml` — `project.title` and `project.repositories`. The SpecScore Repo Config Feature explicitly states `project.repositories` is "metadata for downstream tools (e.g., orchestration platforms, viewers)" — i.e., for Synchestra itself. Its Problem Statement also says the legacy `synchestra-spec-repo.yaml` carried "Synchestra-flavored naming that does not apply to SpecScore proper" — meaning SpecScore intentionally superseded synchestra's split-file scheme; the migration on the synchestra side just hasn't happened. Today's `synchestra project init` and `synchestra project new` commands carry this legacy split, and bootstrapping work for `specstudio:init` exposed the cost: documentation, mental model, and CLI surface all duplicate what specscore.yaml already encodes.

## Recommended Direction

Three coordinated changes, treated as one Idea because each is incoherent without the others.

**1. Enhance SpecScore's `repositories` schema (upstream).** Replace the flat URL list with role-tagged objects: `{url, title, comment, roles}` where `roles` is a non-empty list of values drawn from a small enum (e.g., `code`, `state`, `specification`, `docs`, `runner`). Crucially `roles` is a list, not a scalar — a single repo can be both `specification` and `code` (the synchestra repo is exactly this; specscore likewise). Existing flat-string entries are interpreted as `{url: <string>, roles: [code]}` for backward compatibility during migration. This is a SpecScore-side schema enhancement to the existing `repo-config` Feature.

**2. Synchestra adopts a dedicated `synchestra.yaml` and reads project identity from `specscore.yaml`.** Synchestra introduces `synchestra.yaml` at the repo root, mirroring the SpecScore pattern: a line-1 schema-pointer comment, a single authoritative root file, no front-matter, an empty file is valid. `synchestra.yaml` holds **only synchestra-owned metadata**: state-repo location, sync policy, hub registration ID, runner pinning, etc. Project identity (`title`, `host`, `org`, `repo`, `repositories`) is **read** from `specscore.yaml` — never duplicated. The three legacy `synchestra-*-repo.yaml` files are deprecated; their content splits between the new `synchestra.yaml` (synchestra-only state) and `specscore.yaml` (project identity, code-repo list via role-tagged `repositories`). State repos — which are not specscore-managed and have an independent lifecycle — retain a single `synchestra-state.yaml` self-identifier (renamed from `synchestra-state-repo.yaml`) carrying the back-reference to spec repos.

**3. CLI consolidation around `synchestra init`.** `synchestra project init` and `synchestra project new` collapse into a single top-level `synchestra init` command. The top-level placement deliberately breaks Synchestra's strict noun-verb convention for the bootstrap entry-point — the same exception every CLI ecosystem makes (`git init`, `npm init`, `cargo init`). `synchestra init` has one job: ensure `synchestra.yaml` exists and project state is set up for the specscore.yaml at the current root, in one of three modes — embedded (orphan branch in this repo), separate-repo (state lives at `<url>`), or Hub-managed. Legacy `project init` and `project new` become aliases that dispatch into the unified command, then deprecated.

## Alternatives Considered

**Top-level `synchestra init` as a thin alias for today's `project init`** — keeps the three-file legacy intact; just adds a discoverable verb. *Lost because* the duplication of `title` and `project.repositories` between specscore.yaml and synchestra-spec-repo.yaml is the load-bearing problem. A top-level alias paints over the duplication instead of removing it.

**Embed synchestra-only fields as extension keys inside `specscore.yaml`** — leverage the `unknown-fields-preserved` REQ to add `state_repo:`, `sync:`, `hub:` at the specscore.yaml top level, eliminating the synchestra-*-repo.yaml files entirely. Initially attractive because it minimizes file count. *Lost because* it re-mixes the layers we just spent the conversation pulling apart: SpecScore owns spec-time concerns, Synchestra owns orchestration; one file holding both blurs the boundary. It also creates a flat-namespace collision risk (today's `state_repo:` extension; tomorrow's SpecScore-defined `state_repo:` at the spec level; another orchestrator wanting the same key) that a dedicated synchestra.yaml structurally avoids. The `unknown-fields-preserved` REQ exists primarily for *cross-tool interop hints*, not for an entire orchestrator's config to live as someone else's extension keys.

**Invert the dependency: SpecScore imports project definition from `synchestra-spec-repo.yaml`.** Could fix the duplication too. *Lost because* SpecScore is the broader, more-adopted spec format, and `unknown-fields-preserved` was specifically designed so orchestration tools layer on top of specscore.yaml — not the reverse. Inverting regresses modularity and forces specscore-only adopters to know about synchestra naming.

**Half-collapse: keep three synchestra-* files but add a `synchestra:` block inside specscore.yaml mirroring their content.** *Lost because* it doubles the storage rather than collapsing it — the duplication of `title` / `repositories` is now inside specscore.yaml itself. The point is one source of truth, not synchronized copies.

## MVP Scope

Two timeboxed phases. **Phase 1 (≤2 weeks):** SpecScore schema enhancement landed (role-tagged `repositories` with `roles` list, backward-compat for flat-string entries). `synchestra.yaml` schema defined (line-1 schema-pointer comment, synchestra-only fields documented). Synchestra reads project identity from specscore.yaml AND state config from synchestra.yaml in `info` and `init`, with fallback to legacy `synchestra-spec-repo.yaml` when present. Existing `project init` keeps working unchanged. **Phase 2 (≤2 weeks):** `synchestra init` ships top-level, writing `synchestra.yaml` (embedded mode only — separate-repo and Hub-managed modes deferred). Three-file legacy emits deprecation warnings; not yet removed. The MVP is "not embarrassing" the day one user runs `synchestra init` in a fresh repo with an existing `specscore.yaml` and ends up with exactly two config files at root: `specscore.yaml` (project identity) and `synchestra.yaml` (orchestration state).

## Not Doing (and Why)

- Removing legacy synchestra-* YAML files — keep them readable through Phase 2 with deprecation warnings; deletion is a follow-up Idea/Feature.
- Hub registration as part of `synchestra init` — separate Idea; `synchestra init` Phase 2 stops at embedded-only mode.
- Renaming `~/.synchestra.yaml` user-level config — orthogonal.
- Multi-project repos (multiple `specscore.yaml` roots in one git repo) — out of scope; specscore's `projects:` field handles cross-project navigation.
- `synchestra migrate` CLI command for converting legacy three-file projects to specscore.yaml — referenced as future companion but not specified in this Idea.
- Generalising the typed-`repositories` shape onto the `projects:` cross-project navigation list — possible future, but not this Idea.
- Redesigning the `synchestra project` command group's other verbs (`info`, `set`, `code add/remove`) — they'll need to read from specscore.yaml in Phase 1, but their command surface stays put. A separate Idea may flatten the group later.

## Key Assumptions to Validate

| Tier | Assumption | How to validate |
|------|------------|-----------------|
| Must-be-true | SpecScore maintainers accept the role-tagged `repositories` schema enhancement (multi-valued `roles` list, with `{url, title, comment, roles}` shape) as an additive change to the `repo-config` Feature. | Open a SpecScore Idea/Feature draft for the schema enhancement; gate Phase 1 on its approval. |
| Must-be-true | All existing capability of `synchestra-spec-repo.yaml` and `synchestra-code-repo.yaml` can be cleanly partitioned between (a) specscore.yaml's `project` block (identity, repos) and (b) the new `synchestra.yaml` (orchestration state), with no information loss. | Field-by-field partition mapping in the Feature spec; produce a converted two-file example for each existing test fixture before code lands. |
| Must-be-true | State repos can self-identify with a single state-only file (renamed `synchestra-state.yaml`) — they are not specscore-managed and have an independent lifecycle. | Inspect existing state-repo.yaml usages; confirm none rely on specscore-only fields. Document in the Feature spec. |
| Must-be-true | A dedicated `synchestra.yaml` (parallel to specscore.yaml) is structurally cleaner than embedding synchestra fields as specscore.yaml extension keys, given the namespace-collision and layer-mixing concerns. | Reviewed in the Alternatives section; revisit only if a concrete need emerges to share keys across the boundary. |
| Should-be-true | The `roles` enum stays small and enumerable (≤6 values: code, state, specification, docs, runner, plus reserve). | RFC the enum at spec time; lock at Feature approval. |
| Should-be-true | A repo can be assigned multiple roles and downstream tools handle the union correctly (e.g., a repo tagged `[specification, code]` is scanned by code tooling AND consumed by spec tooling). | Cross-test with the source-references Feature and synchestra's code-repo scanner; add an integration test fixture. |
| Should-be-true | Existing synchestra users can run a one-shot conversion (manual checklist or CLI helper) without bespoke per-project work. | Document the migration in the Feature; verify on at least two real synchestra-managed repos. |
| Might-be-true | The same role-tagged shape applies usefully to `projects:` cross-project navigation entries (parent/child/sibling tags). | Defer; revisit if the navigation use case sharpens. |
| Might-be-true | Hard cutover at synchestra v1.0 is acceptable (no dual-read minor-version bridge). | Defer to release-strategy decision when v1.0 timing firms up. |


## SpecScore Integration

- **New Features this would create:**
  - **(specscore repo)** `repo-config` schema enhancement — role-tagged `repositories` entries with `{url, title, comment, roles}` and multi-valued `roles` list. Likely an additive revision to the existing `spec/features/repo-config/` Feature, not a separate Feature.
  - **(synchestra repo)** `repo-config` (or similar) — Feature defining the `synchestra.yaml` schema: line-1 schema-pointer comment, synchestra-only fields (state-repo, sync policy, hub registration, etc.), how it composes with specscore.yaml's project identity. Mirrors the SpecScore `repo-config` shape at the orchestration layer.
  - **(synchestra repo)** `cli/init` — top-level `synchestra init` command; primary CLI surface for project bootstrap. Writes `synchestra.yaml`.
  - **(synchestra repo)** `state-repo-config` — Feature describing the (renamed) `synchestra-state.yaml` self-identifier file living in state repos.
- **Existing Features affected:**
  - **(specscore repo)** `repo-config` — schema revision (additive backward-compat).
  - **(synchestra repo)** `cli/project/init` — collapses into `cli/init`; kept as deprecated alias in Phase 2.
  - **(synchestra repo)** `cli/project/new` — collapses into `cli/init` (separate-repo mode); kept as deprecated alias.
  - **(synchestra repo)** `cli/project/info` — reads project identity from specscore.yaml + state config from synchestra.yaml in Phase 1, with fallback to legacy synchestra-spec-repo.yaml.
  - **(synchestra repo)** `cli/project/set` — writes synchestra-only fields to synchestra.yaml; identity edits go to specscore.yaml.
  - **(synchestra repo)** `cli/project/code/add` and `code/remove` — operate on `project.repositories` in specscore.yaml (the canonical code-repo list).
- **Dependencies:**
  - SpecScore schema enhancement (role-tagged `repositories`) is a hard prerequisite for Phase 1 of the synchestra-side work.
  - `specstudio:init` skill (already shipped) calls `synchestra init` in its Step 6; that consumer becomes real once Phase 2 lands and now writes `synchestra.yaml` rather than synchestra extension keys inside specscore.yaml.
  - The `unknown-fields-preserved` REQ in SpecScore stays as a cross-tool interop hint, not as the primary integration point — this Idea explicitly de-emphasises it.

## Outstanding Questions

- **Final `roles` enum.** Starting set is `code`, `state`, `specification`, `docs`, `runner`. May also need `sandbox`, `data`, `infrastructure`, `proposal`. Lock at Feature spec time.
- **`synchestra.yaml` schema authority.** Where does the canonical `synchestra.yaml` schema live and at what URL? The line-1 schema-pointer comment must point somewhere — `https://synchestra.md/repo-config` or analogous. Decide at Feature spec time.
- **Migration strategy.** Hard cutover at synchestra v1.0 versus dual-read for one minor version then drop. Tied to v1.0 release timing.
- **`synchestra init` mode-selection mechanism.** Three modes (embedded / separate-repo / Hub-managed). Wizard-driven, flag-driven, or both? Defer to the `cli/init` Feature spec.
- **Whether `cli/project` group survives the consolidation.** After `init` and `new` flatten, the group only holds `info`, `set`, `code add/remove`. Worth flattening too? Separate Idea.
- **Synchestra's reaction to a code repo without its own specscore.yaml.** A code-only repo participating in a synchestra project may not be specscore-managed. Does it still need a `synchestra.yaml` of its own (with a `roles: [code]` self-tag), or is its membership encoded only in the spec repo's `project.repositories` list? Probably the latter, but worth resolving in the Feature spec.
- ~~**Whether state repos use `synchestra.yaml` or a distinct `synchestra-state.yaml`.**~~ **Resolved 2026-05-08:** state repos use a distinct `synchestra-state.yaml` filename. Provisional choice — may revisit if separate-repo / Hub-managed modes surface a stronger argument for filename unification. Tracked by the future `state-repo-config` Feature.

---
*This document follows the https://specscore.md/idea-specification*
