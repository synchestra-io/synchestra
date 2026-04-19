# Feature: Agent Skills

**Status:** In Progress

## Summary

A set of resource-level skills that AI agents use to interact with Synchestra — one skill per CLI resource group (`task`, `feature`, `runner`, `session`, …) with per-verb instructions loaded on demand via Claude Code's progressive-disclosure mechanism. Skills expose *when* to call the CLI, *what* to run, and *how* to interpret results, while keeping the slash-menu surface scannable.

## Problem

AI agents (Claude Code, Cursor, Windsurf, etc.) need a structured way to interact with Synchestra during their work. Without skills, agents would need to:

- Know the full CLI syntax from memory or documentation
- Handle error codes and retry logic ad-hoc
- Guess when to call Synchestra vs. continue working

Skills solve this by providing machine-readable instructions that agent platforms can load and invoke at the right moment.

## Design Principles

### One skill per CLI resource, one reference per action

Each skill covers a single Synchestra CLI resource group (`task`, `feature`, `runner`, `session`, `auth`, `project`, `spec`, `code`). The skill's `SKILL.md` is a thin **index** — a table mapping user intent to the correct reference file. Per-verb instructions live in `references/<verb>.md` and are loaded on demand only when the agent needs them.

This replaces the earlier "one skill per CLI command" rule. The resource-level grouping keeps the slash menu scannable (~10 entries instead of ~34) while preserving 1:1 traceability between CLI commands and reference files. See [ADR-0002](../../decisions/0002-progressive-disclosure-skills.md) for the decision record.

Examples of resource-level skills:

- `task` — covers all task verbs (claim, start, status, complete, fail, release, abort, block, unblock, enqueue, new, info, list)
- `feature` — covers feature queries (list, tree, info, deps, refs, new)
- `runner` — covers remote runner management and `dispatch`
- `session` — covers runtime session inspection
- `auth` — covers user-to-Hub authentication
- `whats-next` — meta skill that helps agents pick their next task

### Plugin namespace provides the prefix

Claude Code automatically prefixes a plugin's skills with the plugin manifest's `name` field. For the `synchestra-cli` plugin (repository [`ai-plugin-synchestra`](https://github.com/synchestra-io/ai-plugin-synchestra)), skills are invoked as `/synchestra-cli:<skill-name>`.

Skill directory names **must not repeat** the plugin namespace. A directory named `synchestra-task/` inside a plugin named `synchestra-cli` would render as `/synchestra-cli:synchestra-task` — a double prefix. The correct form is `task/` → `/synchestra-cli:task`.

Skill frontmatter `name:` values, if set, must match the directory name and use lowercase letters, digits, and hyphens only (no colons). See [ADR-0003](../../decisions/0003-skill-naming-plugin-namespace.md) for the decision record.

### Selective menu visibility

Not every resource-level skill should appear in the human's `/` slash menu. Most Synchestra skills are agent-facing — the human invokes the `synchestra` CLI directly from a terminal, while the skill exists so Claude can use it during autonomous work. Only a subset (daily-workflow entry points, conversational exploration) is worth exposing to human invocation.

Each resource-level `SKILL.md` declares `user-invocable:` **explicitly** in its frontmatter:

- `user-invocable: true` — human naturally types `/synchestra-cli:<name>` to invoke. Daily-workflow and conversational skills.
- `user-invocable: false` — primarily agent-driven. Claude still auto-invokes based on the description; the skill is simply hidden from the `/` menu.

The flag affects menu visibility only. Descriptions still load into context, auto-invocation still works, full `SKILL.md` still loads when dispatched. See [ADR-0005](../../decisions/0005-user-invocable-visibility.md) for the classification rule and initial assignments.

### Skills wrap the CLI

Skills are not an alternative to the CLI — they wrap it. Each reference file (`references/<verb>.md`) gives the agent:

- **When to use it** — trigger conditions
- **What to run** — the exact CLI command with parameter descriptions
- **What happens next** — exit code interpretation and follow-up actions

### Exit code contract

All Synchestra CLI commands follow a consistent exit code contract:

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `1` | Conflict (e.g., another agent claimed first) |
| `2` | Invalid arguments |
| `3` | Resource not found |
| `4` | Invalid state transition |
| `10+` | Unexpected errors |

Command-group-specific ranges (e.g., `40–49` for `task`) are documented in [`spec/features/cli/README.md`](../cli/README.md#exit-code-contract). On non-zero exit, the CLI writes a human-readable explanation to stderr.

## Progressive Disclosure

Claude Code skills load in three tiers. Synchestra skills exploit all three to keep token cost low:

| Tier | Loaded when | Contents |
|---|---|---|
| 1. Frontmatter `description` | Always (at plugin install / session start) | One-line routing hint per resource-level skill |
| 2. `SKILL.md` body | When the Skill tool is invoked | Index table: intent → reference file |
| 3. `references/<verb>.md` | When the agent follows a markdown link via the Read tool | Full instructions for one CLI verb |

Only tier 1 is always in the agent's context. Tier 2 loads on skill invocation. Tier 3 loads only when the agent explicitly reads the file after picking the right verb from the index. This is how a plugin with 34 CLI verbs fits comfortably under 10 slash-menu entries without sacrificing per-verb detail.

## Skill File Format

Skills live in the dedicated [`ai-plugin-synchestra`](https://github.com/synchestra-io/ai-plugin-synchestra) repository, published as the `synchestra-cli` plugin via the `synchestra-io` Claude Code marketplace. Each resource-level skill is a directory containing `SKILL.md` and a `references/` subdirectory with per-verb files.

```
ai-plugin-synchestra/
  skills/
    README.md                   ← skills index, vision, and available skills table
    task/
      SKILL.md                  ← index table for all task verbs
      references/
        claim.md
        start.md
        status.md
        complete.md
        fail.md
        release.md
        abort.md
        block.md
        unblock.md
        new.md
        enqueue.md
        info.md
        list.md
    feature/
      SKILL.md
      references/
        list.md
        tree.md
        info.md
        deps.md
        refs.md
        new.md
    runner/
    session/
    auth/
    project/
    spec/
    code/
    whats-next/
```

### `SKILL.md` structure (the index)

Each resource-level `SKILL.md` contains:

- **Frontmatter** — `description:` (required), `name:` (optional, matches directory name)
- **Brief body** — one or two sentences identifying the resource
- **Index table** — intent phrasing in the left column, reference link in the right column

The index table uses **user intent** as the left-column phrasing, not verb names. "Reserve a queued task for yourself" is preferred over "claim" because it matches how the agent reasons about the action.

### `references/<verb>.md` structure

Each per-verb reference file contains the full instructions for one CLI command:

- **When to use** — trigger conditions for the agent
- **Command** — the CLI invocation with parameters (link to canonical `_args/` spec)
- **Parameters** — description of each flag
- **Exit codes** — what each code means and what the agent should do
- **Examples** — concrete usage

Reference files do not have their own frontmatter or plugin-namespace exposure. They are plain markdown loaded by the agent via the Read tool.

## Distribution

Skills are distributed to agents through:

- **Claude Code plugin install:** `/plugin install synchestra-cli@synchestra-io` after adding the [`synchestra-io` marketplace](https://github.com/synchestra-io/ai-marketplace)
- **Synchestra CLI:** `synchestra skill list` and `synchestra skill show <name>` for on-demand access (reads from the locally installed plugin or the published repository)
- **MCP server:** skills exposed as MCP tools that agents on other platforms can discover and call
- **Direct file access:** agents can read skills directly from the [`ai-plugin-synchestra`](https://github.com/synchestra-io/ai-plugin-synchestra/tree/main/skills) repository or a local install

## Plans

- [Agent Skills Roadmap](../../plans/agent-skills-roadmap/README.md) — phased plan for building out navigation, mutation, and workflow skills

See the [skills README](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/README.md) for the full list of available skills, the vision for how skills transform agent workflows, and token cost analysis.

## Related Decisions

- [ADR-0001](../../decisions/0001-extract-ai-plugin.md) — AI plugin extracted to a dedicated repository
- [ADR-0002](../../decisions/0002-progressive-disclosure-skills.md) — progressive-disclosure skill structure
- [ADR-0003](../../decisions/0003-skill-naming-plugin-namespace.md) — skill directory names must not repeat the plugin namespace
- [ADR-0004](../../decisions/0004-layered-plugin-architecture.md) — layered plugin architecture (CLI wrappers + methodology plugins)
- [ADR-0005](../../decisions/0005-user-invocable-visibility.md) — per-resource `user-invocable` visibility

## Outstanding Questions

- Should skills include platform-specific instructions (e.g., "in Claude Code, add this to your CLAUDE.md")?
- How are skills versioned? Does the CLI version imply the skill version, or are they independent?
- Should there be a machine-readable skill manifest (e.g., `skill.yaml`) alongside the README, or is the README sufficient?
- Should the canonical `SKILL.md` index-table format be stricter (e.g., required column headers, enforced phrasing conventions) so index tables stay consistent across resource skills?
