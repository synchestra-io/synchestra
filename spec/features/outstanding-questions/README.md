# Feature: Outstanding Questions

**Status:** Conceptual

## Summary

Every document in Synchestra maintains an "Outstanding questions" section. Questions can be linked to tasks; when the task completes, questions can be automatically resolved. Recently resolved questions remain visible briefly for context before being archived.

## Problem

When AI agents work on tasks, they frequently encounter ambiguities, unknowns, and decisions that need human input. Today, these get lost in chat logs, buried in commit messages, or forgotten between sessions. The next agent (or the same agent in a new session) rediscovers the same unknowns, wastes tokens reasoning about them, and may make inconsistent decisions.

## Proposed Behavior

### Every document has an "Outstanding questions" section

This is a structural requirement enforced by the project schema. If the section is empty, it explicitly states "None at this time." — not omitted, so its absence is always a schema violation.

### Question lifecycle

```
Open → Linked (optional) → Resolved → Recently Resolved → Archived/Removed
```

1. **Open.** A question is added by a human or agent. It sits in the "Outstanding questions" section of the relevant document.

2. **Linked.** A question can be linked to a sub-task (e.g., "Research authentication providers" addresses the question "Which auth provider should we use?"). Linking is optional but enables automatic resolution.

3. **Resolved.** When a linked task completes, a specialized sub-agent evaluates whether the task's output actually answers the question. If yes, the question is marked resolved. Questions can also be resolved manually by a human or agent via the CLI.

4. **Recently resolved.** Resolved questions move to a "Recently resolved questions" section, keeping the last few visible for context. This helps agents and humans see what was recently decided without digging through git history.

5. **Archived/Removed.** After a configurable period or count threshold, resolved questions are removed from the document. Git history preserves them permanently.

### Adding questions

```bash
# Via CLI
synchestra question add --doc spec/features/auth/README.md \
  --text "Which OAuth provider should we use?"

# Or manually — just add to the "Outstanding questions" section
```

### Linking to tasks

```bash
synchestra question link --question Q-123 --task task_research_auth
```

### Auto-resolution

When a linked task completes:
1. The resolution sub-agent reads the task's output artifacts
2. Compares them against the question text
3. If the output addresses the question, resolves it and optionally adds a one-line summary of the answer
4. If unclear, leaves the question open and logs the assessment

## Open Design Decisions

- How are questions identified? Sequential IDs? UUIDs? Content-addressable?
- Should questions be cross-referenced across documents? (e.g., the same question appears in a feature spec and a task)
- How many "recently resolved" questions should be kept? Configurable per-project?
- Should the auto-resolution sub-agent use a specific model class, or inherit from the document's context?

## Outstanding Questions

- What is the schema for a question entry? (Fields: text, status, linked_task, resolved_at, resolution_summary?)
- Can questions have priority or urgency levels?
- Should there be a project-wide view of all outstanding questions across all documents?
