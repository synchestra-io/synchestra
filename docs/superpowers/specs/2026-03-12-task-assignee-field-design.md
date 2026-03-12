# Task Assignee Field

## Summary

Add an `assignee` field to tasks that indicates who the task is for: AI agents, humans, tools, or a combination. Enables collaborative environments where AI agents pick up tasks automatically, humans get notified, and external tools watch for relevant work.

## Field Definition

**Field:** `assignee` (optional, string)

Empty/unset means any worker can claim the task.

### Grammar

```
assignee   = type_expr [ ":" role_expr ]
type_expr  = type [ "|" type ]*
type       = "ai-agent" | "human" | "tool"
role_expr  = role_group [ ";" role_group ]*
role_group = role [ ":" id_list ]
id_list    = id [ "," id ]*
role       = <alphanumeric, hyphens, underscores>
id         = <alphanumeric, hyphens, underscores>
```

### Parsing Algorithm

Given an assignee string, parse as follows:

1. **Split on first `:`** to get `type_expr` and optional `role_expr`
2. **Split `type_expr` on `|`** to get individual types (closed enum: `ai-agent`, `human`, `tool`)
3. **Split `role_expr` on `;`** to get role groups
4. **Split each role group on first `:`** to get `role` and optional `id_list`
5. **Split `id_list` on `,`** to get individual IDs

Example: `ai-agent|human:code_reviewer:alex,bob;owner:jack`
- Step 1: type_expr=`ai-agent|human`, role_expr=`code_reviewer:alex,bob;owner:jack`
- Step 2: types=[`ai-agent`, `human`]
- Step 3: role_groups=[`code_reviewer:alex,bob`, `owner:jack`]
- Step 4: [{role:`code_reviewer`, ids:`alex,bob`}, {role:`owner`, ids:`jack`}]
- Step 5: [{role:`code_reviewer`, ids:[`alex`,`bob`]}, {role:`owner`, ids:[`jack`]}]

### Matching Rules

1. **Types are AND with roles.** `ai-agent|human:code_reviewer` means: (ai-agent OR human) AND (code_reviewer). Both levels must match.
2. **Role groups are OR.** `code_reviewer:alex,bob;owner:jack` means: match `code_reviewer` with id `alex` or `bob`, OR match `owner` with id `jack`.
3. **Values are case-sensitive and whitespace-free.** No spaces around separators. `AI-Agent` is invalid; `ai-agent` is correct.

### Separator Hierarchy

| Level | Separator | Purpose | Example |
|-------|-----------|---------|---------|
| Type/Role boundary | `:` | Separates type from role (`ai-agent:small`) and role from ids (`small:copilot`) | `ai-agent:small` |
| Type | `\|` | OR between actor types | `ai-agent\|human` |
| Role group | `;` | OR between role groups | `code_reviewer;owner` |
| ID | `,` | OR between identities within a role | `alex,bob` |

### Examples

| `assignee` value | Meaning |
|---|---|
| *(empty)* | Anyone can claim |
| `ai-agent` | Any AI agent |
| `ai-agent:small` | AI agent with role "small" |
| `ai-agent:frontend-design:copilot` | AI agent, role "frontend-design", id "copilot" |
| `human` | Any human |
| `human:approver:alex` | Human, role "approver", id "alex" |
| `ai-agent\|human` | AI agent or human, any role |
| `ai-agent\|human:code_reviewer:alex,bob;owner:jack` | (AI agent or human) AND (code_reviewer with id alex or bob, OR owner with id jack) |
| `tool:security-scanner:sonarqube` | Tool, role "security-scanner", id "sonarqube" |

## Claim Enforcement

- **Type: hard-enforced.** CLI rejects claim if claimer's `--actor-type` doesn't match the task's assignee type list.
- **Role & ID: soft/advisory.** Shown in listings, used by orchestrators for routing, not enforced by CLI.
- If assignee is empty/unset, any `--actor-type` is accepted.

### Validation Logic

1. Parse task's `assignee` field
2. If empty/unset: allow any `--actor-type`
3. If type list present: `--actor-type` must match at least one type
4. If no match: reject with exit code 5 and message: `"actor type '<type>' not allowed; task requires: <type_list>"`

**Exit code 5** = assignee type mismatch (distinct from exit code 2 which means malformed arguments).

## Relationship with `agent_id`

The existing `agent_id` field (in the REST API) identifies **who is currently working** on a task — set at claim time. The new `assignee` field specifies **who should work** on a task — set at creation time and used for routing.

- `assignee` = routing constraint (pre-claim)
- `agent_id` / `run` = actual worker identity (post-claim)

They are complementary. `assignee` does not replace `agent_id`.

## CLI Integration

### Task Creation

```bash
synchestra task create my-task --assignee "ai-agent:small"
synchestra task create audit-task --assignee "tool:security-scanner:sonarqube"
synchestra task create review --assignee "ai-agent|human:code_reviewer"
```

### Task Claiming

```bash
synchestra task claim my-task --run 4821 --actor-type ai-agent
synchestra task claim review --run 4821 --actor-type human
```

`--actor-type` is required when the task has a non-empty assignee. If the task's assignee is empty/unset, `--actor-type` is optional (backward compatible).

### Task Reassignment (new command)

```bash
synchestra task reassign my-task --assignee "human:senior-dev"
```

Works in any status except `completed` (exit code 4: invalid transition). Commits and pushes atomically.

### Task Listing (new filters)

```bash
synchestra task list --assignee-type ai-agent
synchestra task list --assignee-role code_reviewer
synchestra task list --assignee-id alex
```

## Reassignment Guards

- **Blocked in:** `completed` (exit code 4)
- **Allowed in:** all other statuses

Note: `aborted` tasks can be re-queued and reassigned.

## Storage

### Task README Frontmatter

```yaml
---
title: Implement login page
status: queued
assignee: ai-agent|human:frontend-design
depends_on: design-system
---
```

### Task Board Table

```markdown
| Task | Status | Assignee | Run | Model |
|------|--------|----------|-----|-------|
| implement-login | queued | ai-agent\|human:frontend-design | | |
| security-audit | in_progress | tool:scanner:sonarqube | 5521 | |
```

New `Assignee` column appears after `Status` and before `Run`.

## Affected Skills & Specs

### Updates Required

- `synchestra-task-create` — add `--assignee` parameter
- `synchestra-claim-task` — add `--actor-type` requirement and type validation
- `synchestra-task-list` — add `--assignee-type`, `--assignee-role`, `--assignee-id` filters
- `synchestra-task-info` — display assignee field

### New

- `synchestra-task-reassign` — reassign a task's assignee field

### API Updates (`docs/api/tasks.md`)

- Add `assignee` to task creation payload (alongside existing `agent_id`)
- Add `actor_type` to claim payload
- Add `assignee` filter params to list endpoint
- New `PATCH /api/v1/tasks/:id/assignee` endpoint for reassignment

### No Changes Needed

- `start`, `complete`, `fail`, `block`, `unblock`, `release`, `abort`, `aborted`, `enqueue`, `status`

## Outstanding Questions

- Should duplicate types (e.g., `ai-agent|ai-agent`) be rejected at parse time or silently deduplicated?
- Should releasing a claimed task preserve or clear the assignee? (Current assumption: preserve)
- Should the `assignee` field be validated at creation time (e.g., reject unknown types), or accept any string and validate only at claim time?
