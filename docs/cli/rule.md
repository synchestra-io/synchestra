# CLI: `synchestra rule`

Create and manage rules — constraints and instructions that shape how agents behave within a given scope.

**See also:** [API: Rules](../api/rules.md) | [Feature: Human Steering](../features/human-steering.md)

---

## Commands

| Subcommand | Description |
|---|---|
| [create](#create) | Create a rule |
| [list](#list) | List rules |
| [get](#get) | Get rule details |

---

## create

Create a rule. Rules are attached to a scope (human, org, project, or repo) and are surfaced to agents working in that context.

```
synchestra rule create [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--name` | ✅ | Short name for the rule |
| `--content` | ✅ | The rule content — a clear, actionable instruction |
| `--scope` | ✅ | Where this rule applies: `human`, `org`, `project`, `repo` |
| `--scope-id` | ✅ | ID of the human/org/project/repo this rule belongs to |

### Examples

```bash
# Project-level rule
synchestra rule create \
  --name "no-direct-prod-deploy" \
  --content "Never deploy directly to production. Always deploy to staging first, run smoke tests, then request human approval before proceeding to production." \
  --scope project \
  --scope-id proj_abc123

# Org-level rule
synchestra rule create \
  --name "no-data-deletion" \
  --content "Never DELETE records from the database. Use soft deletes with an is_deleted flag. This rule applies to all migrations and application code." \
  --scope org \
  --scope-id org_xyz789

# Repo-level coding standard
synchestra rule create \
  --name "test-coverage-required" \
  --content "All new functions must have at least one unit test. Do not submit a task as complete if new code lacks test coverage." \
  --scope repo \
  --scope-id repo_def456

# Human-level preference
synchestra rule create \
  --name "prefer-simplicity" \
  --content "When in doubt between two implementation options, choose the simpler one. No premature optimisation." \
  --scope human \
  --scope-id human_jkl789
```

**See also:** [POST /api/v1/rules](../api/rules.md#create-rule)

---

## list

List rules, optionally filtered by scope.

```
synchestra rule list [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--scope` | Filter by scope type: `human`, `org`, `project`, `repo` |
| `--scope-id` | Filter by specific scope ID |

### Examples

```bash
# All rules
synchestra rule list

# Rules for a specific project
synchestra rule list --scope project --scope-id proj_abc123

# All org-level rules
synchestra rule list --scope org

synchestra rule list --scope project --scope-id proj_abc123 --output json
```

**See also:** [GET /api/v1/rules](../api/rules.md#list-rules)

---

## get

Get details for a specific rule.

```
synchestra rule get <id> [flags]
```

### Examples

```bash
synchestra rule get rule_abc123
```

**See also:** [GET /api/v1/rules/:id](../api/rules.md#get-rule)
