# ADR-0002: Progressive-disclosure skill structure

**Status:** Approved
**Date:** 2026-04-19

## Context

The `ai-plugin-synchestra` plugin ships 25 skills today, each wrapping exactly one `synchestra` CLI command (`synchestra-task-claim`, `synchestra-feature-info`, etc.). Nine more are queued for runner/session/auth — about 34 total once they land.

The [`agent-skills` feature spec](../features/agent-skills/README.md) currently encodes *one skill per action*: each skill maps to exactly one CLI command, in its own directory with its own `README.md`.

Two concrete pains surfaced as the count grew:

1. **Slash-menu scale.** The primary invocation UX is a human typing `/` in Claude Code. 34 entries under a single plugin prefix is unscannable — autocomplete narrows by prefix, but with 14 task verbs alone, muscle memory breaks down.
2. **Convention drift already happened.** Four of the 25 existing skills use `<verb>-<resource>` (e.g., `synchestra-claim-task`) while the rest use `<resource>-<verb>`. The inconsistency is direct evidence that the convention was never properly encoded.

Claude Code skills support **progressive disclosure** natively: a skill's `SKILL.md` body is loaded when the Skill tool is invoked, but files under `references/` are only loaded when the agent follows a markdown link via the Read tool. The sister plugin [`ai-plugin-sdd`](https://github.com/synchestra-io/ai-plugin-sdd) already uses this pattern (`specscore-design/references/`).

## Decision

Restructure Synchestra skills as **resource-level skills with per-verb references**.

- One skill per CLI resource group (`synchestra-task`, `synchestra-feature`, `synchestra-runner`, `synchestra-session`, `synchestra-auth`, `synchestra-project`, `synchestra-spec`, `synchestra-code`), plus standalone skills for meta operations like `synchestra-whats-next`.
- Each resource-level skill has a `SKILL.md` that is an **index** — a thin table mapping user intent to the correct reference file. No per-verb instructions live in `SKILL.md`.
- Per-verb instructions live in `references/<verb>.md` (e.g., `synchestra-task/references/claim.md`). The agent reads them on demand via the Read tool by following a markdown link from the index.

Target structure:

```
ai-plugin-synchestra/skills/
├── synchestra-task/
│   ├── SKILL.md                 # index of 14 task verbs
│   └── references/
│       ├── claim.md
│       ├── start.md
│       ├── status.md
│       └── ...
├── synchestra-feature/
│   ├── SKILL.md
│   └── references/
│       └── ...
├── synchestra-runner/
├── synchestra-session/
├── synchestra-auth/
├── synchestra-project/
├── synchestra-spec/
├── synchestra-code/
└── synchestra-whats-next/
```

Three-tier loading model:

| Tier | Loaded when | Contents |
|---|---|---|
| 1. Frontmatter `description` | Always (skill discovery) | One-line routing hint, per resource |
| 2. `SKILL.md` body | Skill tool invoked | Index table: intent → reference |
| 3. `references/<verb>.md` | Agent follows link via Read | Full instructions for one CLI verb |

Skill names retain the `synchestra-` prefix (not `synchestra:`). The colon-namespace question is out of scope for this ADR and may be revisited once Claude Code's slash-menu grouping semantics are verified.

## Consequences

**Easier**

- The `/` menu drops from ~34 entries to ~10. Human slash-command UX becomes scannable.
- Token cost per invocation drops: the agent loads a thin index plus one targeted reference, not the full-detail SKILL.md for every skill.
- The 1:1 CLI traceability contract is preserved — it moves from skill level to reference level.
- Adding a new CLI verb means adding one file under `references/`, not creating a new skill directory. Ecosystem growth has less friction.
- The inversion bug (`synchestra-claim-task` vs `synchestra-task-claim`) is retired during the migration.

**Harder**

- The index table becomes a critical routing surface. If intent phrasing is ambiguous, the agent opens the wrong reference and wastes a round trip.
- Slightly more indirection: the agent makes a Skill tool invocation and then a Read, instead of one Skill invocation that contains everything.
- Not symmetric with `ai-plugin-sdd`, where each skill is a single workflow. The shape diverges because Synchestra's skills are resource-oriented rather than workflow-oriented.

**Mitigations**

- Intent phrasing in the index table is written as user intent ("Reserve a queued task for yourself") rather than verb name ("claim"). This reduces routing-error rate.
- The full per-verb instructions remain authoritative in `references/<verb>.md`; no information is lost in the collapse.
- The updated `agent-skills` spec will define a canonical index-table shape so new resource skills stay consistent.

## Alternatives considered

1. **Status quo + fix inversion.** Keep one skill per CLI verb; rename the four offenders; add runner/session/auth as new top-level skills. Rejected — addresses the inconsistency bug but leaves 34 entries in the slash menu, which is the stated primary pain.
2. **Hierarchical namespace (`synchestra:task:claim`).** Rejected for now — relies on Claude Code's slash menu handling two-colon names gracefully, which is unverified. Also keeps all 34 skills present; autocomplete narrowing is the only scale mitigation.
3. **Workflow-collapsed skills (`synchestra-work`, `synchestra-progress`, `synchestra-finish`, …).** Rejected — cuts the count to ~8 but loses 1:1 CLI traceability. Each skill becomes a fat chooser with internal branching, increasing routing-error risk and maintenance cost when a single verb changes.
4. **Hybrid common-vs-rare (1:1 for common verbs, one collapsed skill per resource for rare verbs).** Rejected — two patterns coexist in one plugin. Every new verb raises "is this common or rare?" as a judgment call, which calcifies into bikeshedding.

## Follow-ups

- Update [`spec/features/agent-skills/README.md`](../features/agent-skills/README.md) to encode the resource-level + references shape, document the three-tier loading model, and retire the "one skill per action" framing.
- Migrate the existing 25 skills in `ai-plugin-synchestra` to the new structure. Bump plugin version (0.0.1 → 0.1.0) to signal the breaking change.
- Write the new runner/session/auth skills as resource-level skills from the start.
- Define the canonical `SKILL.md` index-table format in the updated feature spec (header row, intent-column phrasing, link format).
- Revisit the `synchestra-` vs `synchestra:` prefix question as a separate ADR once Claude Code's slash-menu namespace behavior is understood.

---
*This document follows the https://specscore.md/decision-specification*
