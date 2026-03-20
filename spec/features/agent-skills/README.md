# Feature: Agent Skills

**Status:** In Progress

## Summary

A set of dedicated, focused skills that AI agents use to interact with Synchestra — claiming tasks, reporting status, updating progress, and more. Each skill wraps a single Synchestra CLI command with clear trigger conditions, parameters, and exit code handling.

## Problem

AI agents (Claude Code, Cursor, Windsurf, etc.) need a structured way to interact with Synchestra during their work. Without skills, agents would need to:

- Know the full CLI syntax from memory or documentation
- Handle error codes and retry logic ad-hoc
- Guess when to call Synchestra vs. continue working

Skills solve this by providing machine-readable instructions that agent platforms can load and invoke at the right moment.

## Design Principles

### One skill per action

Each skill maps to exactly one Synchestra CLI command. No multi-purpose skills. This keeps skills small, testable, and easy to reason about.

Examples of individual skills:
- `synchestra-claim-task` — claim a task for work
- `synchestra-report-status` — report progress on a claimed task
- `synchestra-complete-task` — mark a task as complete
- `synchestra-fail-task` — mark a task as failed with reason
- `synchestra-release-task` — release a claimed task back to the queue
- `synchestra-list-tasks` — list available tasks

### Skills wrap the CLI

Skills are not an alternative to the CLI — they wrap it. The skill provides the agent with:
- **When to use it** — trigger conditions
- **What to run** — the exact CLI command with parameter descriptions
- **What happens next** — exit code interpretation and follow-up actions

### Exit code contract

All Synchestra CLI commands follow a consistent exit code contract:

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `1` | Claim conflict (another agent claimed first) |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Invalid state transition (e.g., completing an unclaimed task) |
| `10+` | Unexpected errors |

On non-zero exit, the CLI writes a human-readable explanation to stderr.

## Skill File Format

Skills live in `ai-plugin/skills/{skill-name}/README.md` in the main Synchestra repository.

```
ai-plugin/skills/
  README.md                       ← skills index, vision, and available skills table
  synchestra-claim-task/
    README.md
  synchestra-feature-info/
    README.md
  ...
```

Each skill README follows a consistent structure:
- **Name and description** — what the skill does
- **When to use** — trigger conditions for the agent
- **Command** — the CLI invocation with parameters
- **Parameters** — description of each flag
- **Exit codes** — what each code means and what the agent should do
- **Examples** — concrete usage

## Distribution

Skills are distributed to agents through:
- **Synchestra CLI:** `synchestra skills list` and `synchestra skills show <name>` for on-demand access
- **MCP server:** Skills exposed as MCP tools that agents can discover and call
- **Direct file access:** Agents working in the Synchestra repo can read skills directly from `ai-plugin/skills/`

## Plans

- [Agent Skills Roadmap](../../plans/agent-skills-roadmap/README.md) — phased plan for building out navigation, mutation, and workflow skills

See the [skills README](../../../ai-plugin/skills/README.md) for the full list of available skills, the vision for how skills transform agent workflows, and token cost analysis.

## Outstanding Questions

- Should skills include platform-specific instructions (e.g., "in Claude Code, add this to your CLAUDE.md")?
- How are skills versioned? Does the CLI version imply the skill version, or are they independent?
- Should there be a machine-readable skill manifest (e.g., `skill.yaml`) alongside the README, or is the README sufficient?
