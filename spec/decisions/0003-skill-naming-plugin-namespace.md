# ADR-0003: Skill directory names must not repeat the plugin namespace

**Status:** Approved
**Date:** 2026-04-19

## Context

[ADR-0002](0002-progressive-disclosure-skills.md) decided the **shape** of Synchestra skills (resource-level skills with per-verb `references/`). It left the **naming convention** for skill directories and frontmatter `name:` fields as an open follow-up, pending verification of how Claude Code renders plugin-shipped skills.

Official Claude Code documentation confirms the behavior. Two direct quotes:

> "`name` — Unique identifier and skill namespace. **Skills are prefixed with this** (e.g., `/my-first-plugin:hello`)."
> — [Plugin manifest docs](https://code.claude.com/docs/en/plugins)

> "**Plugin skills use a `plugin-name:skill-name` namespace**, so they cannot conflict with other levels."
> — [Skills docs](https://code.claude.com/docs/en/skills)

The plugin manifest's `name` field is automatically the skill namespace. There is no opt-out other than renaming the plugin itself. The same rule applies to commands and agents shipped by plugins.

The Synchestra plugin's manifest name is `synchestra-cli`. Existing skill directories inside `ai-plugin-synchestra` are named `synchestra-task-claim`, `synchestra-feature-info`, etc. Under current Claude Code rules, these render as `/synchestra-cli:synchestra-task-claim` — the `synchestra-` prefix is duplicated. The same issue affects the sister `ai-plugin-sdd` plugin (name `spec-driven-development` + skill directory `specscore-design` renders as `/spec-driven-development:specscore-design`).

A second constraint from the same spec:

> "`name` — Display name for the skill. If omitted, uses the directory name. **Lowercase letters, numbers, and hyphens only (max 64 characters).**"

The frontmatter `name:` field permits only lowercase letters, digits, and hyphens. Colons are invalid. The existing `ai-plugin-sdd` frontmatter uses `name: specscore:design`, which violates this rule and silently falls back to the directory name.

## Decision

Skill directory names and skill frontmatter `name:` values **must not repeat the plugin manifest's `name`**. The plugin name is already the namespace — repeating it creates a double prefix.

Concrete rules for the `synchestra-cli` plugin (repository `ai-plugin-synchestra`):

1. Skill directories are named after the resource only: `task/`, `feature/`, `runner/`, `session/`, `auth/`, `project/`, `spec/`, `code/`, `whats-next/`. No `synchestra-` prefix.
2. Skill frontmatter `name:` is omitted (directory name wins) or set to match the directory name. Always lowercase + hyphens only. No colons.
3. Users invoke skills as `/synchestra-cli:<skill-name>`, for example `/synchestra-cli:task`. Claude Code constructs the prefix automatically.
4. Per-verb files under `references/` follow the same rule — lowercase + hyphens, no plugin-name prefix. Example: `task/references/claim.md`, not `task/references/synchestra-task-claim.md`.

This rule applies to any future Synchestra-authored plugin (see [ADR-0001](0001-extract-ai-plugin.md) for the `ai-plugin-*` pattern). A plugin named `specscore-cli` would have skills like `plan/`, `lint/`, etc., never `specscore-plan/`.

## Consequences

**Easier**

- Slash-menu invocation is clean: `/synchestra-cli:task`, not `/synchestra-cli:synchestra-task`.
- Skill names are shorter by 11 characters (`synchestra-` prefix removed).
- The rule is mechanical — no judgment call per skill.
- Cross-plugin consistency becomes a single pattern: `<plugin-name>:<resource>`.
- Matches what the Claude Code docs show in every example.

**Harder**

- One-time migration of the existing 25 skills in `ai-plugin-synchestra` (rename directories, update frontmatter, bump plugin version 0.0.1 → 0.1.0). Already required by ADR-0002.
- The `ai-plugin-sdd` plugin has pre-existing skill frontmatter (`name: specscore:design`) that violates the lowercase-hyphens rule. Needs a separate cleanup pass, but that is outside this ADR's scope.
- Documentation examples across the Synchestra repository that reference old skill names (`synchestra-claim-task`, etc.) become stale and need replacement.

## Alternatives considered

1. **Keep the `synchestra-` prefix on skill directories.** Rejected — produces `/synchestra-cli:synchestra-task`, which is double-prefixed and harder to type. Contradicts every documented Claude Code example.
2. **Rename the plugin from `synchestra-cli` to something shorter (e.g., `sy`).** Rejected — the plugin name is user-visible in `/plugin install <name>@synchestra-io` and in the marketplace. Short plugin names risk collision across the ecosystem; `synchestra-cli` clearly scopes what the plugin wraps.
3. **Use colons in the frontmatter `name:` field (e.g., `name: task:claim`).** Rejected — violates the documented character set (lowercase + hyphens only). The frontmatter value silently falls back to the directory name, so this only creates confusion.

## Follow-ups

- Migrate all 25 existing skills in `ai-plugin-synchestra` as part of the ADR-0002 migration. Rename directories (`synchestra-task-claim/` → `task/`, etc.) and restructure under the resource + references pattern in the same pass.
- Update [`spec/features/agent-skills/README.md`](../features/agent-skills/README.md) to cite ADR-0002 and ADR-0003 and to use the new naming in all examples.
- Flag the `ai-plugin-sdd` frontmatter-validity issue (`name: specscore:design`) to the plugin owner as a separate cleanup task. Not in scope for this repository.
- Close the ADR-0002 follow-up line about the `synchestra-` vs `synchestra:` prefix question; this ADR resolves it.

---
*This document follows the https://specscore.md/decision-specification*
