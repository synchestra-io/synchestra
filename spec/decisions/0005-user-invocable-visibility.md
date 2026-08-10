# ADR-0005: Per-resource `user-invocable` visibility

**Status:** Approved
**Date:** 2026-04-19

## Context

[ADR-0002](0002-progressive-disclosure-skills.md) adopted the resource-level skill structure (one `SKILL.md` per CLI resource, with per-verb detail under `references/`). Under the resulting layout, `ai-plugin-synchestra` exposes roughly 10 resource-level skills in Claude Code's `/` slash menu (`task`, `feature`, `runner`, `session`, `auth`, `project`, `spec`, `code`, `whats-next`, etc.).

A review concern: most of these resources are **agent-facing**. A human sitting in a Claude Code chat is unlikely to ever type `/synchestra-cli:runner` or `/synchestra-cli:session` — those operations are invoked by agents mid-workflow, or by users via the CLI in a terminal. Only a subset of resources (task triage, feature navigation, "what should I work on next") map to how humans interact with Claude Code conversationally. Leaving all resource skills user-invocable pollutes the `/` menu with entries nobody ever picks.

A radical alternative — collapsing every skill into a single mega-skill with two-level nested references — was evaluated and rejected: it doubles routing cost per agent invocation, weakens automatic description matching (one vague description vs. ten targeted ones), and contradicts Claude Code's own "keep `SKILL.md` under 500 lines" guidance.

Claude Code provides a purpose-built frontmatter field. From the [skills reference](https://code.claude.com/docs/en/skills#frontmatter-reference):

> `user-invocable` — Set to `false` to hide from the `/` menu. Use for background knowledge users shouldn't invoke directly. Default: `true`.
>
> "The `user-invocable` field only controls menu visibility, not Skill tool access. Use `disable-model-invocation: true` to block programmatic invocation."

A skill with `user-invocable: false` stays fully agent-accessible: its description loads into context, Claude auto-invokes it when relevant, and the full `SKILL.md` loads on invocation. The human simply doesn't see it in the slash menu.

## Decision

Every resource-level skill in `ai-plugin-synchestra` declares `user-invocable` **explicitly** in its frontmatter. The default (`true`) is not relied on; every skill is classified at authoring time.

Classification criteria:

- **`user-invocable: true`** — a skill a human working in Claude Code would naturally type `/synchestra-cli:<name>` to invoke. Typically: daily-workflow entry points and conversational exploration.
- **`user-invocable: false`** — a skill primarily driven by agents during autonomous work. Humans still get the same behavior (Claude auto-invokes based on the description), but explicit `/` invocation is removed from the menu.

Initial classification for `ai-plugin-synchestra`:

| Skill | `user-invocable` | Rationale |
|---|---|---|
| `whats-next` | `true` | The daily entry point: "what should I work on?" Human-first workflow. |
| `task` | `true` | Humans list tasks, check claimed-task status, and pick work interactively. |
| `feature` | `true` | "LSP for specifications" — humans navigate specs conversationally (list, tree, info, deps, refs). |
| `project` | `false` | Rare, usually once per project. Auto-invocation suffices ("set up a synchestra project"). |
| `spec` | `false` | `spec lint` is CI-land; `spec search` is conversational — auto-invoked by Claude. |
| `code` | `false` | Niche source-to-spec introspection; arises in chat naturally. |
| `runner` (future) | `false` | Remote dispatch; set up once, driven by agents. |
| `session` (future) | `false` | Runtime state inspection — agent-facing. |
| `auth` (future) | `false` | `login` happens once from the CLI; not a menu action. |

Menu surface becomes **3 visible resource skills** (plus `whats-next`) rather than ~10. Agent routing quality is unchanged; only the human-facing menu is pruned.

Future resource skills must be classified explicitly with the same lens.

## Consequences

**Easier**

- The `/` slash menu shows only skills a human would actually type — no pollution.
- Agent behavior is identical to the fully-visible variant: descriptions still load into context, auto-invocation still works, full `SKILL.md` still loads when dispatched.
- Reversible — each classification is one frontmatter flip. No structural migration needed.
- No change to skill shape; `user-invocable` is purely metadata.

**Harder**

- Every new resource skill needs an explicit classification decision. Forgetting flags it as user-invocable (the default), which may over-expose it.
- The classification is a judgment call — "which resources do humans use?" shifts with product surface. Periodic review expected.

## Alternatives considered

1. **Leave all resource skills user-invocable (ADR-0002 as-written).** Rejected — produces menu pollution exactly as described. Violates the original concern that motivated this ADR.
2. **Collapse to a single mega-skill with two-level nested `references/`.** Rejected — doubles routing cost per agent invocation, weakens automatic description matching (one vague description vs. ten targeted ones), and contradicts Claude Code's own length guidance for `SKILL.md`.
3. **Use `disable-model-invocation: true` instead.** Rejected — that flag removes the skill from Claude's *auto-invocation* and from context. It is the opposite of what we want: we want agents to use the skill automatically, we just don't want the human's `/` menu cluttered.

## Follow-ups

- Amend [`spec/features/agent-skills/README.md`](../features/agent-skills/README.md) with a brief `Selective menu visibility` section documenting the classification rule and linking here.
- Apply the classification above during the `ai-plugin-synchestra` migration to the resource + references structure (part of the ADR-0002 / ADR-0003 migration work).
- For every new resource skill (runner, session, auth at least), set `user-invocable` explicitly at authoring time.

---
*This document follows the https://specscore.md/decision-specification*
